package editor

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
)

// AudioClip is an independently positioned, non-destructive audio segment.
// Start is in movie time; SourceStart/SourceEnd are in the source file.
// Audio clips can overlap, and material past the movie's end is not exported.
type AudioClip struct {
	ID             string  `json:"id"`
	Path           string  `json:"path"`
	Name           string  `json:"name"`
	Start          float64 `json:"start"`
	Track          int     `json:"track,omitempty"` // 0..2: separate visual/audio lanes
	SourceStart    float64 `json:"source_start"`
	SourceEnd      float64 `json:"source_end"`
	SourceDuration float64 `json:"source_duration"`
	Volume         float64 `json:"volume"` // 0..2; 1 = original loudness
	FadeIn         float64 `json:"fade_in,omitempty"`
	FadeOut        float64 `json:"fade_out,omitempty"`
	Muted          bool    `json:"muted,omitempty"`
}

func (a AudioClip) Duration() float64 { return a.SourceEnd - a.SourceStart }
func (a AudioClip) Valid() bool {
	return a.Path != "" && isFinite(a.Start) && a.Start >= 0 && a.Track >= 0 && a.Track <= 2 &&
		isFinite(a.SourceStart) && a.SourceStart >= 0 &&
		isFinite(a.SourceEnd) && isFinite(a.SourceDuration) &&
		a.SourceDuration >= MinClipLength && a.Duration() >= MinClipLength-1e-6 && a.SourceEnd <= a.SourceDuration+1e-5 &&
		isFinite(a.Volume) && a.Volume >= 0 && a.Volume <= 2 &&
		isFinite(a.FadeIn) && a.FadeIn >= 0 && a.FadeIn <= a.Duration()+1e-5 &&
		isFinite(a.FadeOut) && a.FadeOut >= 0 && a.FadeOut <= a.Duration()+1e-5
}

// ClipAudioVolume treats old project files (without an audio_volume setting)
// as 100%, while supporting 0% volume without a separate special case.
func (c Clip) ClipAudioVolume() float64 {
	if c.AudioVolume == nil {
		return 1
	}
	return *c.AudioVolume
}

func (p *Project) AddAudio(a AudioClip) error {
	if !a.Valid() {
		return errors.New("invalid audio source, position, volume or fades")
	}
	p.AudioClips = append(p.AudioClips, a)
	return nil
}

func (p *Project) RemoveAudio(index int) error {
	if index < 0 || index >= len(p.AudioClips) {
		return errors.New("invalid audio clip")
	}
	p.AudioClips = append(p.AudioClips[:index], p.AudioClips[index+1:]...)
	return nil
}

func (p *Project) MoveAudio(index int, start float64) error {
	if index < 0 || index >= len(p.AudioClips) || !isFinite(start) {
		return errors.New("invalid audio position")
	}
	p.AudioClips[index].Start = math.Max(0, start)
	return nil
}

// Dragging the left edge keeps the right edge in the same movie position.
func (p *Project) TrimAudioLeft(index int, delta float64) error {
	if index < 0 || index >= len(p.AudioClips) || !isFinite(delta) {
		return errors.New("invalid audio trim")
	}
	a := &p.AudioClips[index]
	old := a.SourceStart
	a.SourceStart = clamp(old+delta, 0, a.SourceEnd-MinClipLength)
	shift := a.SourceStart - old
	if a.Start+shift < 0 {
		a.SourceStart = old - a.Start
		a.Start = 0
	} else {
		a.Start += shift
	}
	a.FadeIn = math.Min(a.FadeIn, a.Duration())
	a.FadeOut = math.Min(a.FadeOut, a.Duration())
	return nil
}

func (p *Project) TrimAudioRight(index int, delta float64) error {
	if index < 0 || index >= len(p.AudioClips) || !isFinite(delta) {
		return errors.New("invalid audio trim")
	}
	a := &p.AudioClips[index]
	a.SourceEnd = clamp(a.SourceEnd+delta, a.SourceStart+MinClipLength, a.SourceDuration)
	a.FadeIn = math.Min(a.FadeIn, a.Duration())
	a.FadeOut = math.Min(a.FadeOut, a.Duration())
	return nil
}

// DetachAudio removes audio from the selected video segment and adds an
// independently movable copy at its current movie time. Video is untouched.
func (p *Project) DetachAudio(index int, newID string) error {
	if index < 0 || index >= len(p.Clips) {
		return errors.New("select a video clip")
	}
	c := &p.Clips[index]
	if !c.HasAudio || c.AudioMuted {
		return errors.New("selected video has no attached audio")
	}
	a := AudioClip{ID: newID, Path: c.Path, Name: filepath.Base(c.Name) + " (detached)",
		Start: p.Start(index), SourceStart: c.SourceStart, SourceEnd: c.SourceEnd, SourceDuration: c.SourceDuration,
		Volume: c.ClipAudioVolume()}
	if !a.Valid() {
		return fmt.Errorf("cannot detach invalid audio clip")
	}
	c.AudioMuted = true
	p.AudioClips = append(p.AudioClips, a)
	return nil
}

// A cached preview proxy must include nontrivial audio edits as well as
// transitions, otherwise the live soundtrack won't match MP4 export.
func (p Project) NeedsRenderedPreview() bool {
	if p.HasTransitions() || len(p.AudioClips) > 0 {
		return true
	}
	for _, c := range p.Clips {
		if c.AudioMuted || c.ClipAudioVolume() != 1 {
			return true
		}
	}
	return false
}
