package editor

import (
	"encoding/json"
	"math"
	"testing"
)

func keyframeClip() Clip {
	return Clip{ID: "source", Path: "video.mp4", SourceDuration: 10, SourceEnd: 10, Title: "Hello"}
}

func TestTitleKeyframeInterpolationAndPersistence(t *testing.T) {
	c := keyframeClip()
	first := TitleKeyframe{Time: 2, X: 0.2, Y: 0.3, Scale: 0.75, Opacity: 0.1}
	last := TitleKeyframe{Time: 8, X: 0.8, Y: 0.9, Scale: 2, Opacity: 0.9}
	if !c.PutTitleKeyframe(last) || !c.PutTitleKeyframe(first) {
		t.Fatal("could not insert")
	}
	for _, test := range []struct{ at, want float64 }{{0, .2}, {2, .2}, {5, .5}, {8, .8}, {10, .8}} {
		got, ok := c.TitleAt(test.at)
		if !ok || math.Abs(got.X-test.want) > 1e-9 {
			t.Fatalf("time %v: %+v", test.at, got)
		}
	}
	if c.TitleKeyframes[0].Time != 2 {
		t.Fatal("not sorted")
	}
	p := NewProject()
	p.Clips = []Clip{c}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	copy := p.Clone()
	copy.Clips[0].TitleKeyframes[0].X = .99
	if p.Clips[0].TitleKeyframes[0].X != .2 {
		t.Fatal("shared keyframe slice in undo clone")
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var loaded Project
	if err = json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if err = loaded.Validate(); err != nil {
		t.Fatal(err)
	}
	if loaded.Clips[0].TitleKeyframes[1] != last {
		t.Fatal("json lost motion")
	}
	c.TitleKeyframes[0].X = .45
	if !c.PutTitleKeyframe(TitleKeyframe{Time: 2.006, X: .55, Y: .3, Scale: 1, Opacity: 1}) || len(c.TitleKeyframes) != 2 {
		t.Fatal("near duplicate instead of replace")
	}
	if !c.DeleteTitleKeyframe(2) || len(c.TitleKeyframes) != 1 {
		t.Fatal("delete")
	}
}
func TestTitleKeyframesSplitTrim(t *testing.T) {
	c := keyframeClip()
	c.PutTitleKeyframe(TitleKeyframe{Time: 0, X: .1, Y: .5, Scale: 1, Opacity: 1})
	c.PutTitleKeyframe(TitleKeyframe{Time: 10, X: .9, Y: .5, Scale: 2, Opacity: .2})
	p := NewProject()
	p.Clips = []Clip{c}
	if _, err := p.SplitAt(4, "second"); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	left, _ := p.Clips[0].TitleAt(4)
	right, _ := p.Clips[1].TitleAt(0)
	if math.Abs(left.X-.42) > 1e-6 || math.Abs(right.X-.42) > 1e-6 {
		t.Fatalf("discontinuous split %v %v", left.X, right.X)
	}
	if err := p.TrimLeft(1, 2); err != nil {
		t.Fatal(err)
	}
	if err := p.TrimRight(0, -1); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	pose, _ := p.Clips[1].TitleAt(0)
	if math.Abs(pose.X-.58) > 1e-6 {
		t.Fatalf("incorrect trim offset: %v", pose.X)
	}
}
func TestTitleKeyframeValidation(t *testing.T) {
	c := keyframeClip()
	for _, bad := range []TitleKeyframe{{Time: 11, X: .5, Y: .5, Scale: 1, Opacity: 1}, {Time: 0, X: 1.4, Y: .5, Scale: 1, Opacity: 1}, {Time: 0, X: .5, Y: .5, Scale: 1, Opacity: math.NaN()}, {Time: 0, X: .5, Y: .5, Scale: 0, Opacity: 1}} {
		if c.PutTitleKeyframe(bad) {
			t.Fatalf("accepted %+v", bad)
		}
	}
	c.TitleKeyframes = []TitleKeyframe{{Time: 1, X: .5, Y: .5, Scale: 1, Opacity: 1}, {Time: .5, X: .5, Y: .5, Scale: 1, Opacity: 1}}
	if c.Valid() {
		t.Fatal("accepted unsorted keyframes")
	}
}
