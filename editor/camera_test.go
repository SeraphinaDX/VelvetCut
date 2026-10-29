package editor

import (
	"encoding/json"
	"math"
	"testing"
)

func cameraClip() Clip {
	return Clip{ID: "movie", Name: "movie", Path: "movie.mp4", SourceDuration: 10, SourceEnd: 10}
}

func TestVideoMotionInterpolationPersistenceAndUndo(t *testing.T) {
	c := cameraClip()
	first := VideoKeyframe{Time: 0, X: 0, Y: .1, Zoom: 1}
	last := VideoKeyframe{Time: 10, X: 1, Y: .9, Zoom: 3}
	if !c.PutVideoKeyframe(last) || !c.PutVideoKeyframe(first) {
		t.Fatal("insert failed")
	}
	pose, ok := c.VideoAt(5)
	if !ok || math.Abs(pose.X-.5) > 1e-6 || math.Abs(pose.Zoom-2) > 1e-6 {
		t.Fatalf("wrong middle interpolation: %+v", pose)
	}
	p := NewProject()
	p.Clips = []Clip{c}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	original := p.Clone()
	original.Clips[0].VideoKeyframes[0].X = .25
	if p.Clips[0].VideoKeyframes[0].X != 0 {
		t.Fatal("undo shares keyframes")
	}
	data, _ := json.Marshal(p)
	var loaded Project
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if err := loaded.Validate(); err != nil {
		t.Fatal(err)
	}
	if loaded.Clips[0].VideoKeyframes[1] != last {
		t.Fatal("lost serialized video motion")
	}
	if !c.PutVideoKeyframe(VideoKeyframe{Time: .005, X: .1, Y: .1, Zoom: 1.5}) || len(c.VideoKeyframes) != 2 {
		t.Fatal("failed replace")
	}
	if !c.DeleteVideoKeyframe(0) || len(c.VideoKeyframes) != 1 {
		t.Fatal("failed removal")
	}
}

func TestVideoMotionSplitAndTrim(t *testing.T) {
	c := cameraClip()
	c.PutVideoKeyframe(VideoKeyframe{Time: 0, X: 0, Y: .5, Zoom: 1})
	c.PutVideoKeyframe(VideoKeyframe{Time: 10, X: 1, Y: .5, Zoom: 3})
	p := NewProject()
	p.Clips = []Clip{c}
	if _, err := p.SplitAt(4, "second"); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	left, _ := p.Clips[0].VideoAt(4)
	right, _ := p.Clips[1].VideoAt(0)
	if math.Abs(left.X-.4) > 1e-5 || math.Abs(right.X-.4) > 1e-5 {
		t.Fatalf("split jumped: %+v %+v", left, right)
	}
	if err := p.TrimLeft(1, 2); err != nil {
		t.Fatal(err)
	}
	pose, _ := p.Clips[1].VideoAt(0)
	if math.Abs(pose.X-.6) > 1e-5 {
		t.Fatalf("trim jumped: %+v", pose)
	}
	if err := p.TrimRight(0, -1); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestVideoMotionInvalidData(t *testing.T) {
	c := cameraClip()
	for _, bad := range []VideoKeyframe{{Time: 11, X: .5, Y: .5, Zoom: 1}, {Time: 0, X: 1.1, Y: .5, Zoom: 1}, {Time: 0, X: .5, Y: .5, Zoom: 0}, {Time: 0, X: .5, Y: .5, Zoom: math.NaN()}} {
		if c.PutVideoKeyframe(bad) {
			t.Fatalf("accepted %+v", bad)
		}
	}
	c.VideoKeyframes = []VideoKeyframe{{Time: 2, X: .5, Y: .5, Zoom: 1}, {Time: 1, X: .5, Y: .5, Zoom: 1}}
	if c.Valid() {
		t.Fatal("accepted unsorted video keyframes")
	}
}

// Camera-only edits also require a composited playback proxy; otherwise
// mpv could play the original unanimated video while preview/export are right.
func TestCameraMotionTriggersRenderedPreview(t *testing.T) {
	p := NewProject()
	c := cameraClip()
	p.Clips = []Clip{c}
	if p.NeedsRenderedPreview() {
		t.Fatal("plain clip unexpectedly needs proxy")
	}
	c.PutVideoKeyframe(VideoKeyframe{Time: 0, X: .5, Y: .5, Zoom: 2})
	p.Clips[0] = c
	if !p.NeedsRenderedPreview() {
		t.Fatal("camera keyframes need composited preview")
	}
}
