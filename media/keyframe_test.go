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

// This test exercises drawtext's actual per-frame expression evaluator, not
// just string construction. A flat backdrop makes the moving white title easy
// to detect separately from the source video pixels.
func TestAnimatedTitlePlaybackProxyAndStills(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	input := filepath.Join(t.TempDir(), "flat.mp4")
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "color=c=0x205060:s=320x180:r=15:d=2", "-c:v", "mpeg4", input)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sample: %v %s", err, out)
	}
	c := editor.Clip{ID: "a", Name: "test", Path: input, SourceDuration: 2, SourceEnd: 2, Title: "MOVE", Visual: editor.VisualSettings{TitleSeconds: 2, TitleSize: 88, TitleAnimation: "zoom"}}
	c.PutTitleKeyframe(editor.TitleKeyframe{Time: 0, X: .15, Y: .5, Scale: .7, Opacity: 1})
	c.PutTitleKeyframe(editor.TitleKeyframe{Time: 0.8, X: .50, Y: .4, Scale: 1.0, Opacity: .7})
	c.PutTitleKeyframe(editor.TitleKeyframe{Time: 1.5, X: .85, Y: .5, Scale: 1.3, Opacity: 1})
	p := editor.NewProject()
	p.Clips = []editor.Clip{c}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "proxy.mp4")
	if err := ExportPreview(ctx, p, out); err != nil {
		t.Fatalf("export keyframed title: %v", err)
	}
	for _, sec := range []float64{.1, 1.7} {
		img, err := StyledFrame(ctx, c, sec)
		if err != nil {
			t.Fatalf("at %f: %v", sec, err)
		}
		x := brightestTextCentroid(img)
		if sec < .5 && x > img.Bounds().Dx()/2 {
			t.Fatalf("early title unexpectedly at x=%d", x)
		}
		if sec > 1 && x < img.Bounds().Dx()/2 {
			t.Fatalf("late title unexpectedly at x=%d", x)
		}
	}
}

func brightestTextCentroid(img image.Image) int {
	b := img.Bounds()
	var sum, n int
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r > 0xc000 && g > 0xc000 && bl > 0xc000 {
				sum += x
				n++
			}
		}
	}
	if n == 0 {
		return -1
	}
	return sum / n
}
