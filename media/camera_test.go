package media

import (
	"context"
	"image"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/SeraphinaDX/VelvetCut/editor"
)

// A moving camera must move the actual picture, in both styled stills and
// exported preview video (which shares the filter graph with full MP4 export).
func TestCameraKeyframesRenderAndExport(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	input := filepath.Join(t.TempDir(), "halves.mp4")
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "color=c=red:s=320x180:r=15:d=2", "-vf", "drawbox=x=160:y=0:w=160:h=180:color=blue:t=fill", "-c:v", "mpeg4", input)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sample: %v %s", err, out)
	}
	clip := editor.Clip{ID: "sample", Name: "sample", Path: input, SourceDuration: 2, SourceEnd: 2}
	clip.PutVideoKeyframe(editor.VideoKeyframe{Time: 0, X: 0, Y: .5, Zoom: 2})
	clip.PutVideoKeyframe(editor.VideoKeyframe{Time: 1.5, X: 1, Y: .5, Zoom: 2})
	p := editor.NewProject()
	p.Clips = []editor.Clip{clip}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		at   float64
		blue bool
	}{{.1, false}, {1.85, true}} {
		im, err := StyledFrame(ctx, clip, tc.at)
		if err != nil {
			t.Fatalf("styled frame at %.2f: %v", tc.at, err)
		}
		if got := isBlueCentre(im); got != tc.blue {
			t.Fatalf("frame at %.2fs blue=%t want %t", tc.at, got, tc.blue)
		}
	}
	output := filepath.Join(t.TempDir(), "proxy.mp4")
	if err := ExportPreview(ctx, p, output); err != nil {
		t.Fatalf("export: %v", err)
	}
	for _, tc := range []struct {
		at   float64
		blue bool
	}{{.1, false}, {1.85, true}} {
		im, err := Frame(ctx, output, tc.at)
		if err != nil {
			t.Fatal(err)
		}
		if got := isBlueCentre(im); got != tc.blue {
			t.Fatalf("export at %.2fs blue=%t want %t", tc.at, got, tc.blue)
		}
	}
}

func isBlueCentre(im image.Image) bool {
	b := im.Bounds()
	r, _, blue, _ := im.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2).RGBA()
	return blue > r*2
}
