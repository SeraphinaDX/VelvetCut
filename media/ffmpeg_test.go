package media

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/SeraphinaDX/VelvetCut/editor"
)

func TestExportMixedSoundAndSilentClips(t *testing.T) {
	if err := CheckTools(); err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	withAudio := filepath.Join(dir, "test with spaces.mp4")
	silent := filepath.Join(dir, "silent.mp4")
	mkVideo := func(path string, audio bool) {
		args := []string{"-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "color=c=blue:s=320x180:r=30:d=1.2"}
		if audio {
			args = append(args, "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=1.2")
		}
		args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p")
		if audio {
			args = append(args, "-c:a", "aac", "-shortest")
		}
		args = append(args, path)
		if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
			t.Fatalf("generate video: %v: %s", err, out)
		}
	}
	mkVideo(withAudio, true)
	mkVideo(silent, false)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	metaA, err := Probe(ctx, withAudio)
	if err != nil || !metaA.HasAudio {
		t.Fatalf("probe sound: %+v %v", metaA, err)
	}
	metaB, err := Probe(ctx, silent)
	if err != nil || metaB.HasAudio {
		t.Fatalf("probe silent: %+v %v", metaB, err)
	}
	p := editor.NewProject()
	_ = p.Insert(0, editor.Clip{ID: "a", Path: withAudio, Name: "a", SourceStart: 0.1, SourceEnd: 0.7, SourceDuration: metaA.Duration, HasAudio: true})
	_ = p.Insert(1, editor.Clip{ID: "b", Path: silent, Name: "b", SourceStart: 0.2, SourceEnd: 0.8, SourceDuration: metaB.Duration})
	output := filepath.Join(dir, "exported.mp4")
	if err := Export(ctx, p, output); err != nil {
		t.Fatal(err)
	}
	if stat, err := os.Stat(output); err != nil || stat.Size() < 500 {
		t.Fatalf("no output: %v", err)
	}
	result, err := Probe(ctx, output)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasAudio {
		t.Fatal("sound track was lost")
	}
	if math.Abs(result.Duration-1.2) > 0.22 {
		t.Fatalf("export duration %v, expected roughly 1.2", result.Duration)
	}
	frame, err := Frame(ctx, output, 0.5)
	if err != nil || frame.Bounds().Dx() == 0 {
		t.Fatalf("frame: %v", err)
	}
}

func TestExportTitleAndFades(t *testing.T) {
	if err := CheckTools(); err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "source.mp4")
	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "color=c=blue:s=320x180:r=30:d=1.5", "-f", "lavfi", "-i", "sine=frequency=260:duration=1.5", "-shortest", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", input}
	if output, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("setup: %s: %v", output, err)
	}
	p := editor.NewProject()
	p.Clips = []editor.Clip{{ID: "a", Path: input, SourceStart: 0, SourceEnd: 1.2, SourceDuration: 1.5, HasAudio: true, Title: "Girl's cut: 100% ready!", FadeIn: 0.2, FadeOut: 0.3}}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	destination := filepath.Join(dir, "title.mp4")
	if err := Export(ctx, p, destination); err != nil {
		t.Fatal(err)
	}
	meta, err := Probe(ctx, destination)
	if err != nil || !meta.HasAudio || math.Abs(meta.Duration-1.2) > 0.15 {
		t.Fatalf("probe: %+v %v", meta, err)
	}
	peaks, err := Waveform(ctx, input, 1.5)
	if err != nil || len(peaks) < 10 {
		t.Fatalf("waveform: %v: %d samples", err, len(peaks))
	}
	var max float32
	for _, v := range peaks {
		if v > max {
			max = v
		}
	}
	if max <= 0.01 {
		t.Fatalf("waveform unexpectedly quiet: %.6f", max)
	}
}
