// Package editor contains VelvetCut's UI-independent, non-destructive edit model.
package editor

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
)

const MinClipLength = 0.15

var ErrOutsideClip = errors.New("playhead must be inside a clip, away from its edges")

type Clip struct {
	ID             string  `json:"id"`
	Path           string  `json:"path"`
	Name           string  `json:"name"`
	SourceStart    float64 `json:"source_start"`
	SourceEnd      float64 `json:"source_end"`
	SourceDuration float64 `json:"source_duration"`
	HasAudio       bool    `json:"has_audio"`
	Title          string  `json:"title,omitempty"`
	FadeIn         float64 `json:"fade_in,omitempty"`
	FadeOut        float64 `json:"fade_out,omitempty"`
}

func (c Clip) Duration() float64 { return c.SourceEnd - c.SourceStart }

func (c Clip) Valid() bool {
	return c.Path != "" && isFinite(c.SourceStart) && isFinite(c.SourceEnd) &&
		isFinite(c.SourceDuration) && c.SourceDuration >= MinClipLength &&
		c.SourceStart >= 0 && c.Duration() >= MinClipLength-1e-6 &&
		c.SourceEnd <= c.SourceDuration+1e-5 &&
		isFinite(c.FadeIn) && isFinite(c.FadeOut) && c.FadeIn >= 0 && c.FadeOut >= 0 &&
		c.FadeIn <= c.Duration()+1e-5 && c.FadeOut <= c.Duration()+1e-5
}

func isFinite(f float64) bool { return !math.IsInf(f, 0) && !math.IsNaN(f) }

type Project struct {
	Version int    `json:"version"`
	Clips   []Clip `json:"clips"`
}

func NewProject() Project { return Project{Version: 1, Clips: []Clip{}} }

func (p Project) Clone() Project {
	clone := p
	clone.Clips = append([]Clip(nil), p.Clips...)
	return clone
}

func (p Project) Duration() float64 {
	var total float64
	for _, c := range p.Clips {
		total += c.Duration()
	}
	return total
}

func (p Project) Start(index int) float64 {
	var start float64
	for i := 0; i < index && i < len(p.Clips); i++ {
		start += p.Clips[i].Duration()
	}
	return start
}

// At translates a position in the edited movie into a source media timestamp.
func (p Project) At(t float64) (index int, sourceTime float64, ok bool) {
	if len(p.Clips) == 0 || t < 0 {
		return 0, 0, false
	}
	total := p.Duration()
	if t > total+1e-6 {
		return 0, 0, false
	}
	if t >= total {
		t = math.Max(0, total-1e-5)
	}
	elapsed := 0.0
	for i, c := range p.Clips {
		next := elapsed + c.Duration()
		if t < next {
			return i, c.SourceStart + t - elapsed, true
		}
		elapsed = next
	}
	return 0, 0, false
}

func (p *Project) Insert(index int, c Clip) error {
	if !c.Valid() {
		return errors.New("invalid clip source/duration")
	}
	if index < 0 || index > len(p.Clips) {
		return errors.New("invalid insertion index")
	}
	p.Clips = append(p.Clips, Clip{})
	copy(p.Clips[index+1:], p.Clips[index:])
	p.Clips[index] = c
	return nil
}

// InsertAtTime performs ripple insertion at an exact timeline position. If the
// playhead falls inside a clip, that clip is split before inserting the media.
// Positions too close to an existing cut snap to that cut.
func (p *Project) InsertAtTime(t float64, c Clip, splitID string) (int, error) {
	if !c.Valid() || !isFinite(t) || t < 0 || t > p.Duration()+1e-5 {
		return -1, errors.New("invalid clip or insert position")
	}
	if len(p.Clips) == 0 || t >= p.Duration() {
		return len(p.Clips), p.Insert(len(p.Clips), c)
	}
	elapsed := 0.0
	for i, old := range p.Clips {
		next := elapsed + old.Duration()
		if t <= elapsed+MinClipLength {
			return i, p.Insert(i, c)
		}
		if t >= next-MinClipLength && t <= next {
			return i + 1, p.Insert(i+1, c)
		}
		if t < next {
			if _, err := p.SplitAt(t, splitID); err != nil {
				return -1, err
			}
			return i + 1, p.Insert(i+1, c)
		}
		elapsed = next
	}
	return len(p.Clips), p.Insert(len(p.Clips), c)
}

// SplitAt splits the clip underlying the movie timeline position.
func (p *Project) SplitAt(t float64, newID string) (int, error) {
	index, sourceTime, ok := p.At(t)
	if !ok {
		return -1, ErrOutsideClip
	}
	old := p.Clips[index]
	if sourceTime-old.SourceStart < MinClipLength || old.SourceEnd-sourceTime < MinClipLength {
		return -1, ErrOutsideClip
	}
	first, second := old, old
	first.SourceEnd = sourceTime
	second.SourceStart = sourceTime
	first.FadeOut = 0
	second.FadeIn = 0
	first.FadeIn = math.Min(first.FadeIn, first.Duration())
	second.FadeOut = math.Min(second.FadeOut, second.Duration())
	second.ID = newID
	p.Clips[index] = first
	p.Clips = append(p.Clips, Clip{})
	copy(p.Clips[index+2:], p.Clips[index+1:])
	p.Clips[index+1] = second
	return index + 1, nil
}

// TrimLeft moves the start of the source range. Positive delta shortens the clip.
func (p *Project) TrimLeft(index int, delta float64) error {
	if index < 0 || index >= len(p.Clips) || !isFinite(delta) {
		return errors.New("invalid trim")
	}
	c := &p.Clips[index]
	c.SourceStart = clamp(c.SourceStart+delta, 0, c.SourceEnd-MinClipLength)
	c.FadeIn = math.Min(c.FadeIn, c.Duration())
	c.FadeOut = math.Min(c.FadeOut, c.Duration())
	return nil
}

// TrimRight moves the end of the source range. Negative delta shortens the clip.
func (p *Project) TrimRight(index int, delta float64) error {
	if index < 0 || index >= len(p.Clips) || !isFinite(delta) {
		return errors.New("invalid trim")
	}
	c := &p.Clips[index]
	c.SourceEnd = clamp(c.SourceEnd+delta, c.SourceStart+MinClipLength, c.SourceDuration)
	c.FadeIn = math.Min(c.FadeIn, c.Duration())
	c.FadeOut = math.Min(c.FadeOut, c.Duration())
	return nil
}

func (p *Project) Remove(index int) error {
	if index < 0 || index >= len(p.Clips) {
		return errors.New("invalid clip")
	}
	p.Clips = append(p.Clips[:index], p.Clips[index+1:]...)
	return nil
}

// Move reorders a clip to a final index (after removal of the original).
func (p *Project) Move(from, to int) error {
	if from < 0 || from >= len(p.Clips) || to < 0 || to >= len(p.Clips) {
		return errors.New("invalid position")
	}
	clip := p.Clips[from]
	p.Clips = append(p.Clips[:from], p.Clips[from+1:]...)
	p.Clips = append(p.Clips, Clip{})
	copy(p.Clips[to+1:], p.Clips[to:])
	p.Clips[to] = clip
	return nil
}

func (p Project) Validate() error {
	if p.Version != 1 {
		return fmt.Errorf("unsupported project version %d", p.Version)
	}
	for i, c := range p.Clips {
		if !c.Valid() {
			return fmt.Errorf("clip %d (%s) is invalid", i+1, filepath.Base(c.Path))
		}
	}
	return nil
}

func clamp(value, low, high float64) float64 { return math.Max(low, math.Min(high, value)) }
