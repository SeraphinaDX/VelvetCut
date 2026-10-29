package editor

import (
	"encoding/json"
	"math"
	"testing"
)

func TestAudioNonDestructiveEditingAndDetach(t *testing.T) {
	p := NewProject()
	_ = p.Insert(0, sample("scene", 4))
	if err := p.DetachAudio(0, "separate"); err != nil {
		t.Fatal(err)
	}
	if !p.Clips[0].AudioMuted || len(p.AudioClips) != 1 {
		t.Fatal("detach must mute source and add one independent track")
	}
	a := p.AudioClips[0]
	eq(t, a.Start, 0)
	eq(t, a.SourceStart, 0)
	eq(t, a.Duration(), 4)
	if err := p.MoveAudio(0, 1.25); err != nil {
		t.Fatal(err)
	}
	if err := p.TrimAudioLeft(0, 0.5); err != nil {
		t.Fatal(err)
	}
	eq(t, p.AudioClips[0].Start, 1.75)
	eq(t, p.AudioClips[0].SourceStart, 0.5)
	if err := p.TrimAudioRight(0, -0.25); err != nil {
		t.Fatal(err)
	}
	eq(t, p.AudioClips[0].Duration(), 3.25)
	p.AudioClips[0].FadeIn = 0.5
	p.AudioClips[0].Volume = 0.4
	p.AudioClips[0].Track = 2
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	clone := p.Clone()
	clone.AudioClips[0].Volume = 0.1
	if math.Abs(p.AudioClips[0].Volume-0.4) > 1e-8 {
		t.Fatal("clone alias")
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
	if loaded.AudioClips[0].Track != 2 || loaded.AudioClips[0].Volume != 0.4 {
		t.Fatal("audio settings not persisted")
	}
	if err = p.RemoveAudio(0); err != nil || len(p.AudioClips) != 0 {
		t.Fatal("remove audio")
	}
}

func TestOldProjectAudioDefaultsAndInvalidSettings(t *testing.T) {
	p := NewProject()
	_ = p.Insert(0, sample("scene", 4))
	if p.Clips[0].ClipAudioVolume() != 1 {
		t.Fatal("old projects must default to 100% volume")
	}
	p.AudioClips = []AudioClip{{ID: "bad", Path: "music.wav", SourceDuration: 2, SourceEnd: 2, Volume: 1, Track: 3}}
	if p.Validate() == nil {
		t.Fatal("track out of range accepted")
	}
	p.AudioClips[0].Track = 0
	p.AudioClips[0].FadeOut = 3
	if p.Validate() == nil {
		t.Fatal("invalid fade accepted")
	}
	p.AudioClips[0].FadeOut = 1
	if p.Validate() != nil {
		t.Fatal("valid audio rejected")
	}
	if !p.NeedsRenderedPreview() {
		t.Fatal("audio soundtrack requires effects proxy")
	}
}
