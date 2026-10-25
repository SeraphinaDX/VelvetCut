package media

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestStreamFrames(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("requires ffmpeg")
	}
	dir := t.TempDir()
	video := filepath.Join(dir, "sample.mp4")
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=s=160x90:r=20:d=1", "-c:v", "mpeg4", video)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create video: %v: %s", err, output)
	}
	if _, err := os.Stat(video); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	frames, err := StreamFrames(ctx, video, .2, .5, 10)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	for frame := range frames {
		if frame.Image.Bounds().Dx() == 0 {
			t.Fatal("empty frame")
		}
		if count == 0 && frame.SourceTime != .2 {
			t.Fatalf("start=%v", frame.SourceTime)
		}
		count++
	}
	if count < 2 || count > 8 {
		t.Fatalf("unexpected frames: %d", count)
	}
}
