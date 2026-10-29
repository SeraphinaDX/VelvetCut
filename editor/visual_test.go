package editor

import (
	"encoding/json"
	"math"
	"testing"
)

func TestVisualSettingsBackwardCompatibility(t *testing.T) {
	var p Project
	if err := json.Unmarshal([]byte(`{"version":1,"clips":[{"path":"old.mp4","source_start":0,"source_end":4,"source_duration":4}]}`), &p); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if p.NeedsRenderedPreview() {
		t.Fatal("plain old clip should not need a render")
	}
	p.Clips[0].Visual = VisualSettings{Effect: "sepia", Brightness: 0.2, Zoom: 1.3, Rotation: 90}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if !p.NeedsRenderedPreview() {
		t.Fatal("visual edits must request an accurate proxy")
	}
	clone := p.Clone()
	clone.Clips[0].Visual.Effect = "bw"
	if p.Clips[0].Visual.Effect != "sepia" {
		t.Fatal("clone modified source project")
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var round Project
	if err = json.Unmarshal(data, &round); err != nil {
		t.Fatal(err)
	}
	if round.Clips[0].Visual != p.Clips[0].Visual {
		t.Fatal("style metadata did not round trip")
	}
}
func TestVisualSettingsValidation(t *testing.T) {
	for _, v := range []VisualSettings{{Brightness: math.NaN()}, {Effect: "invalid"}, {Rotation: 45}, {Zoom: 5}, {TitleSize: 400}, {TitleSeconds: -2}, {Contrast: -0.8}, {Saturation: -2}} {
		if v.Valid() {
			t.Fatalf("accepted invalid settings %+v", v)
		}
	}
	if !(VisualSettings{}).Valid() {
		t.Fatal("zero-value settings must be valid")
	}
}
