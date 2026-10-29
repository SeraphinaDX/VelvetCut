package media

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/SeraphinaDX/VelvetCut/editor"
)

func TestVisualEffectsExportAndPausedPreview(t *testing.T) {
	for _, name := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(name); err != nil {
			t.Skip(name + " unavailable")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "input.mp4")
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "testsrc2=size=160x90:rate=20:duration=2", "-f", "lavfi", "-i", "sine=frequency=440:duration=2", "-shortest", "-c:v", "mpeg4", "-c:a", "aac", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("setup: %v %s", err, out)
	}
	cases := []struct {
		name   string
		visual editor.VisualSettings
	}{
		{"default-title", editor.VisualSettings{}},
		{"fade-title", editor.VisualSettings{TitleFont: "Mono", TitleColor: "yellow", TitleAnimation: "fade", TitlePosition: "top", TitleSize: 70, TitleSeconds: 1.5}},
		{"slide-title", editor.VisualSettings{TitleFont: "Serif", TitleColor: "cyan", TitleAnimation: "slide", TitlePosition: "center"}},
		{"zoom-title", editor.VisualSettings{TitleColor: "pink", TitleAnimation: "zoom", TitlePosition: "lower-left"}},
		{"rotate-crop", editor.VisualSettings{Rotation: 90, Crop: "square", Zoom: 1.3, Brightness: .2, Contrast: .1, Saturation: -.2, Temperature: .4}},
		{"sepia", editor.VisualSettings{Effect: "sepia", Crop: "fill"}},
		{"bw", editor.VisualSettings{Effect: "bw", Rotation: 180, Crop: "4:3"}},
		{"vignette", editor.VisualSettings{Effect: "vignette", Rotation: 270}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			c := editor.Clip{Path: path, Name: "example", SourceEnd: 2, SourceDuration: 2, HasAudio: true, Title: "Hello, VelvetCut! 100%", Visual: tt.visual}
			p := editor.NewProject()
			p.Clips = []editor.Clip{c}
			if err := p.Validate(); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(t.TempDir(), "render.mp4")
			if err := ExportPreview(ctx, p, output); err != nil {
				t.Fatalf("export: %v", err)
			}
			meta, err := Probe(ctx, output)
			if err != nil || !meta.HasAudio {
				t.Fatalf("missing exported video/audio %v: %+v", err, meta)
			}
			if _, err := StyledFrame(ctx, c, .8); err != nil {
				t.Fatalf("styled still: %v", err)
			}
		})
	}
}
