package media

import (
	"bytes"
	"context"
	"image"
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


// Regression: JPEG image2pipe decoding used to return only the first frame.
// Check that a continuous sample produces multiple different images.
func TestStreamFramesContinuousAndPortrait(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("requires ffmpeg")
	}
	for _, tc := range []struct{ name, size string }{
		{"landscape", "160x90"},
		{"portrait", "90x160"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			video := filepath.Join(t.TempDir(), "movie.mp4")
			cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size="+tc.size+":rate=20:duration=2", "-c:v", "mpeg4", video)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("create video: %v %s", err, out)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			frames, err := StreamFrames(ctx, video, 0, 1, 10)
			if err != nil {
				t.Fatal(err)
			}
			var count int
			var first, last []byte
			for packet := range frames {
				if got := packet.Image.Bounds(); got.Dx() != previewWidth || got.Dy() != previewHeight {
					t.Fatalf("bad dimensions %v", got)
				}
				pix := packet.Image.(*image.RGBA).Pix
				if count == 0 {
					first = append([]byte(nil), pix...)
				}
				last = append(last[:0], pix...)
				count++
			}
			if count < 8 || count > 12 {
				t.Fatalf("expected roughly ten frames, got %d", count)
			}
			if bytes.Equal(first, last) {
				t.Fatal("video frames did not change")
			}
		})
	}
}
