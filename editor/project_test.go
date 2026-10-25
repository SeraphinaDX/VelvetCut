package editor

import (
	"math"
	"testing"
)

func sample(id string, duration float64) Clip {
	return Clip{ID: id, Path: "/video/" + id + ".mp4", Name: id, SourceEnd: duration, SourceDuration: duration, HasAudio: true}
}
func eq(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("got %f, want %f", got, want)
	}
}
func TestSplitKeepsMovieLengthAndMediaRanges(t *testing.T) {
	p := NewProject()
	if err := p.Insert(0, sample("a", 10)); err != nil {
		t.Fatal(err)
	}
	_ = p.Insert(1, sample("b", 4))
	idx, err := p.SplitAt(4, "split")
	if err != nil || idx != 1 {
		t.Fatalf("split: %d %v", idx, err)
	}
	if len(p.Clips) != 3 {
		t.Fatal("incorrect clip count")
	}
	eq(t, p.Duration(), 14)
	eq(t, p.Clips[0].SourceEnd, 4)
	eq(t, p.Clips[1].SourceStart, 4)
	eq(t, p.Clips[1].SourceEnd, 10)
	eq(t, p.Clips[2].SourceStart, 0)
	if _, err = p.SplitAt(4, "nope"); err == nil {
		t.Fatal("allowed split on an edge")
	}
	i, src, ok := p.At(13)
	if !ok || i != 2 {
		t.Fatalf("mapping %d %v", i, ok)
	}
	eq(t, src, 3)
}
func TestTrimClampsToSourceAndMinimum(t *testing.T) {
	p := NewProject()
	_ = p.Insert(0, sample("a", 4))
	if err := p.TrimLeft(0, 1); err != nil {
		t.Fatal(err)
	}
	eq(t, p.Duration(), 3)
	_ = p.TrimRight(0, -1)
	eq(t, p.Duration(), 2)
	_ = p.TrimLeft(0, -100)
	eq(t, p.Clips[0].SourceStart, 0)
	_ = p.TrimRight(0, -100)
	eq(t, p.Clips[0].Duration(), MinClipLength)
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestReorderAndClone(t *testing.T) {
	p := NewProject()
	for _, name := range []string{"a", "b", "c"} {
		_ = p.Insert(len(p.Clips), sample(name, 3))
	}
	before := p.Clone()
	if err := p.Move(0, 2); err != nil {
		t.Fatal(err)
	}
	if p.Clips[0].ID != "b" || p.Clips[2].ID != "a" {
		t.Fatalf("order %#v", p.Clips)
	}
	_ = p.Remove(1)
	if before.Clips[1].ID != "b" || len(before.Clips) != 3 {
		t.Fatal("clone mutated")
	}
}

func TestInsertAtPlayheadSplitsExistingClip(t *testing.T) {
	p := NewProject()
	_ = p.Insert(0, sample("first", 10))
	id, err := p.InsertAtTime(3, sample("inserted", 2), "right-side")
	if err != nil || id != 1 {
		t.Fatalf("insert: %d %v", id, err)
	}
	if len(p.Clips) != 3 || p.Clips[1].ID != "inserted" {
		t.Fatalf("clips: %#v", p.Clips)
	}
	eq(t, p.Duration(), 12)
	eq(t, p.Clips[0].Duration(), 3)
	eq(t, p.Clips[2].SourceStart, 3)
	id, err = p.InsertAtTime(p.Duration(), sample("end", 1), "unused")
	if err != nil || id != 3 {
		t.Fatalf("append: %d %v", id, err)
	}
	eq(t, p.Duration(), 13)
}

func TestTitleFadesSplitAndTrimRemainValid(t *testing.T) {
	p := NewProject()
	c := Clip{ID: "one", Path: "/media/mov.mp4", Name: "mov", SourceStart: 0, SourceEnd: 4, SourceDuration: 8, HasAudio: true, Title: "Cut to the beach", FadeIn: 1, FadeOut: 1}
	if err := p.Insert(0, c); err != nil {
		t.Fatal(err)
	}
	if _, err := p.SplitAt(2, "two"); err != nil {
		t.Fatal(err)
	}
	if p.Clips[0].FadeOut != 0 || p.Clips[1].FadeIn != 0 || p.Clips[0].Title != "Cut to the beach" {
		t.Fatalf("unexpected split effects: %+v", p.Clips)
	}
	if err := p.TrimRight(0, -1.7); err != nil {
		t.Fatal(err)
	}
	if err := p.TrimLeft(1, 1.6); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("invalid after trim: %v: %+v", err, p.Clips)
	}
}
