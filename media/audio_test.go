package media

import (
	"context"
	"encoding/binary"
	"math"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/SeraphinaDX/VelvetCut/editor"
)

func TestAudioTracksMixAndFades(t *testing.T) {
	for _, cmd := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(cmd); err != nil {
			t.Skip(cmd + " unavailable")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	dir := t.TempDir()
	video := filepath.Join(dir, "silent-video.mp4")
	tone := filepath.Join(dir, "song.wav")
	for _, tc := range []struct{ args []string }{
		{[]string{"-f", "lavfi", "-i", "color=c=blue:s=160x90:r=30:d=4", "-c:v", "mpeg4", video}},
		{[]string{"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=3", "-c:a", "pcm_s16le", tone}},
	} {
		cmd := exec.CommandContext(ctx, "ffmpeg", append([]string{"-hide_banner", "-loglevel", "error", "-y"}, tc.args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup: %v %s", err, out)
		}
	}
	duration, err := ProbeAudio(ctx, tone)
	if err != nil || math.Abs(duration-3) > 0.1 {
		t.Fatalf("audio import metadata: %.2f %v", duration, err)
	}
	p := editor.NewProject()
	p.Clips = []editor.Clip{{ID: "video", Path: video, Name: "silent", SourceEnd: 4, SourceDuration: 4}}
	p.AudioClips = []editor.AudioClip{{ID: "song", Name: "song", Path: tone, SourceStart: 0, SourceEnd: 2, SourceDuration: 3, Start: 1, Volume: 0.8, FadeIn: 0.3, FadeOut: 0.4, Track: 1}}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "mix.mp4")
	if err := ExportPreview(ctx, p, out); err != nil {
		t.Fatal(err)
	}
	meta, err := Probe(ctx, out)
	if err != nil {
		t.Fatal(err)
	}
	if !meta.HasAudio || math.Abs(meta.Duration-4) > 0.2 {
		t.Fatalf("wrong output metadata: %+v", meta)
	}
	pcm, err := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-i", out, "-vn", "-ac", "1", "-ar", "8000", "-f", "s16le", "pipe:1").Output()
	if err != nil {
		t.Fatal(err)
	}
	rms := func(start, end float64) float64 {
		l := int(start * 8000)
		r := int(end * 8000)
		if r*2 > len(pcm) {
			t.Fatalf("too few audio samples: %d", len(pcm)/2)
		}
		sum := float64(0)
		for i := l; i < r; i++ {
			v := float64(int16(binary.LittleEndian.Uint16(pcm[i*2:]))) / 32768
			sum += v * v
		}
		return math.Sqrt(sum / float64(r-l))
	}
	if got := rms(0.1, 0.7); got > 0.002 {
		t.Fatalf("audio before overlay should be silent, RMS %f", got)
	}
	if got := rms(1.5, 2.3); got < 0.01 {
		t.Fatalf("overlay did not play, RMS %f", got)
	}
	if got := rms(3.3, 3.8); got > 0.002 {
		t.Fatalf("audio after overlay should be silent, RMS %f", got)
	}
	if got := rms(1.01, 1.08); got >= rms(1.5, 1.7) {
		t.Fatalf("fade-in not applied")
	}
}
