// Package media bridges VelvetCut's edit model to FFmpeg and FFprobe.
// Command arguments are passed directly (never through a shell).
package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SeraphinaDX/VelvetCut/editor"
)

type Metadata struct {
	Duration float64
	HasAudio bool
}

func CheckTools() error {
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s not found in PATH: %w", tool, err)
		}
	}
	return nil
}

func Probe(ctx context.Context, path string) (Metadata, error) {
	var response struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			Duration  string `json:"duration"`
		} `json:"streams"`
	}
	data, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration:stream=codec_type,duration", "-of", "json", path).Output()
	if err != nil {
		return Metadata{}, fmt.Errorf("ffprobe %s: %w", filepath.Base(path), err)
	}
	if err = json.Unmarshal(data, &response); err != nil {
		return Metadata{}, err
	}
	duration, _ := strconv.ParseFloat(response.Format.Duration, 64)
	meta := Metadata{Duration: duration}
	hasVideo := false
	for _, s := range response.Streams {
		if s.CodecType == "video" {
			hasVideo = true
			if duration <= 0 {
				duration, _ = strconv.ParseFloat(s.Duration, 64)
				meta.Duration = duration
			}
		}
		if s.CodecType == "audio" {
			meta.HasAudio = true
		}
	}
	if !hasVideo || meta.Duration < editor.MinClipLength {
		return Metadata{}, errors.New("file has no usable video stream or duration")
	}
	return meta, nil
}

func Frame(ctx context.Context, path string, at float64) (image.Image, error) {
	args := []string{"-hide_banner", "-loglevel", "error", "-ss", seconds(at), "-i", path,
		"-frames:v", "1", "-vf", "scale=960:540:force_original_aspect_ratio=decrease",
		"-f", "image2pipe", "-vcodec", "mjpeg", "pipe:1"}
	data, err := exec.CommandContext(ctx, "ffmpeg", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("extract frame: %w", err)
	}
	frame, _, err := image.Decode(bytes.NewReader(data))
	return frame, err
}

func seconds(n float64) string { return strconv.FormatFloat(n, 'f', 6, 64) }

// Export creates one H.264/AAC MP4 with cuts, trims, audio, and normalized sizes.
// Each source input is separately sought, including repeated portions of a file.
func Export(ctx context.Context, project editor.Project, output string) error {
	return exportWithProfile(ctx, project, output, 1280, 720, 30, "veryfast", "20")
}

// ExportPreview builds an effects-accurate 640x360 transition proxy on demand.
// The same filtergraph is used for final export and preview rendering.
func ExportPreview(ctx context.Context, project editor.Project, output string) error {
	return exportWithProfile(ctx, project, output, 640, 360, 15, "ultrafast", "28")
}

func exportWithProfile(ctx context.Context, project editor.Project, output string, width, height, fps int, preset, crf string) error {
	if err := project.Validate(); err != nil {
		return err
	}
	if len(project.Clips) == 0 {
		return errors.New("nothing to export")
	}
	if output == "" {
		return errors.New("output filename is empty")
	}
	if err := CheckTools(); err != nil {
		return err
	}
	for _, clip := range project.Clips {
		if _, err := os.Stat(clip.Path); err != nil {
			return fmt.Errorf("missing source %q: %w", clip.Path, err)
		}
	}
	// Store titles in UTF-8 files instead of embedding untrusted text in a filtergraph.
	// That preserves punctuation, quotes, non-ASCII letters, and literal colons.
	titlesDir, err := os.MkdirTemp("", "velvetcut-titles-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(titlesDir)
	args := []string{"-hide_banner", "-y", "-loglevel", "error"}
	var filters []string

	for i, clip := range project.Clips {
		duration := clip.Duration()
		args = append(args, "-ss", seconds(clip.SourceStart), "-t", seconds(duration), "-i", clip.Path)
		title, err := makeTitleFile(titlesDir, i, clip.Title)
		if err != nil {
			return err
		}
		filters = append(filters, fmt.Sprintf("[%d:v:0]%s[v%d]", i, visualFilters(clip, width, height, fps, title, -1), i))
		var audio string
		if clip.HasAudio && !clip.AudioMuted {
			audio = fmt.Sprintf("[%d:a:0]aresample=48000,apad,atrim=duration=%s,asetpts=PTS-STARTPTS,aformat=sample_fmts=fltp:channel_layouts=stereo", i, seconds(duration))
		} else {
			audio = fmt.Sprintf("anullsrc=r=48000:cl=stereo,atrim=duration=%s,asetpts=PTS-STARTPTS", seconds(duration))
		}
		if clip.HasAudio && !clip.AudioMuted && clip.ClipAudioVolume() != 1 {
			audio += ",volume=" + seconds(clip.ClipAudioVolume())
		}
		if clip.FadeIn > 0 {
			audio += ",afade=t=in:st=0:d=" + seconds(clip.FadeIn)
		}
		if clip.FadeOut > 0 {
			audio += ",afade=t=out:st=" + seconds(duration-clip.FadeOut) + ":d=" + seconds(clip.FadeOut)
		}
		filters = append(filters, audio+fmt.Sprintf("[a%d]", i))

	}
	// Join segments one boundary at a time. A real transition overlaps its
	// neighboring clips by d seconds, while a cut concatenates directly.
	// xfade and acrossfade apply the same duration, preserving A/V sync.
	videoTrack, audioTrack := "[v0]", "[a0]"
	built := project.Clips[0].Duration()
	for i := 1; i < len(project.Clips); i++ {
		d := project.Clips[i-1].TransitionOut.Duration
		v, a := fmt.Sprintf("[joinedv%d]", i), fmt.Sprintf("[joineda%d]", i)
		if d > 0 {
			effect := "fade"
			if project.Clips[i-1].TransitionOut.Kind == "black" {
				effect = "fadeblack"
			}
			filters = append(filters, fmt.Sprintf("%s[v%d]xfade=transition=%s:duration=%s:offset=%s%s", videoTrack, i, effect, seconds(d), seconds(built-d), v))
			filters = append(filters, fmt.Sprintf("%s[a%d]acrossfade=d=%s:c1=tri:c2=tri%s", audioTrack, i, seconds(d), a))
			built += project.Clips[i].Duration() - d
		} else {
			filters = append(filters, fmt.Sprintf("%s%s[v%d][a%d]concat=n=2:v=1:a=1%s%s", videoTrack, audioTrack, i, i, v, a))
			built += project.Clips[i].Duration()
		}
		videoTrack, audioTrack = v, a
	}
	// Mix extra audio overlays after the video/transition timeline has been
	// assembled. Each source owns its source in/out, independent movie start,
	// gain and fades. The first input is the original soundtrack; overlays can
	// overlap one another. Explicit 48kHz stereo avoids amix format surprises.
	if len(project.AudioClips) > 0 {
		for j, a := range project.AudioClips {
			input := len(project.Clips) + j
			args = append(args, "-ss", seconds(a.SourceStart), "-t", seconds(a.Duration()), "-i", a.Path)
			delay := int64(math.Round(a.Start * 1000))
			audio := fmt.Sprintf("[%d:a:0]aresample=48000,aformat=sample_fmts=fltp:channel_layouts=stereo,atrim=duration=%s,asetpts=PTS-STARTPTS,volume=%s", input, seconds(a.Duration()), seconds(a.Volume))
			if a.Muted {
				audio += ",volume=0"
			}
			if a.FadeIn > 0 {
				audio += ",afade=t=in:st=0:d=" + seconds(a.FadeIn)
			}
			if a.FadeOut > 0 {
				audio += ",afade=t=out:st=" + seconds(a.Duration()-a.FadeOut) + ":d=" + seconds(a.FadeOut)
			}
			audio += fmt.Sprintf(",adelay=%d:all=1,apad,atrim=duration=%s[music%d]", delay, seconds(project.Duration()), j)
			filters = append(filters, audio)
		}
		labels := audioTrack
		for j := range project.AudioClips {
			labels += fmt.Sprintf("[music%d]", j)
		}
		filters = append(filters, fmt.Sprintf("%s amix=inputs=%d:duration=first:dropout_transition=0:normalize=0,atrim=duration=%s[mixedaudio]", labels, len(project.AudioClips)+1, seconds(project.Duration())))
		audioTrack = "[mixedaudio]"
	}
	args = append(args, "-filter_complex", strings.Join(filters, ";"), "-map", videoTrack, "-map", audioTrack,
		"-c:v", "libx264", "-preset", preset, "-crf", crf, "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-b:a", "160k", "-movflags", "+faststart", output)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var log bytes.Buffer
	cmd.Stderr = &log
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg export failed: %w\n%s", err, tail(log.String(), 2500))
	}
	return nil
}

func tail(s string, n int) string {
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}
