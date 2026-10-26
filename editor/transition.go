package editor

import (
 "errors"
 "math"
)

// Transition belongs to the outgoing side of a clip; zero-value means a cut.
// Durations overlap neighboring sources rather than adding black frames.
type Transition struct {
 Kind string `json:"kind,omitempty"` // dissolve or black
 Duration float64 `json:"duration,omitempty"`
}

func (t Transition) Active() bool { return t.Kind != "" && t.Duration > 0 }

func (p Project) HasTransitions() bool {
 for i := 0; i+1 < len(p.Clips); i++ {
  if p.Clips[i].TransitionOut.Active() { return true }
 }
 return false
}

// Cut returns the visible midpoint of a cut or overlap after clip i.
func (p Project) Cut(i int) float64 {
 if i < 0 || i >= len(p.Clips)-1 { return p.Duration() }
 return p.Start(i+1)+p.Clips[i].TransitionOut.Duration/2
}

// SetTransition creates/updates the boundary after outgoing clip i.
// Empty kind or zero duration removes the transition.
func (p *Project) SetTransition(i int, kind string, duration float64) error {
 if i < 0 || i+1 >= len(p.Clips) { return errors.New("choose a boundary between two clips") }
 if !isFinite(duration) || duration < 0 { return errors.New("invalid transition duration") }
 if kind == "" || duration == 0 { p.Clips[i].TransitionOut = Transition{}; return nil }
 if kind != "dissolve" && kind != "black" { return errors.New("unknown transition type") }
 if duration < 0.1 { return errors.New("transitions must be at least 0.1 seconds") }
 old := p.Clips[i].TransitionOut
 p.Clips[i].TransitionOut = Transition{Kind: kind, Duration: duration}
 if err := p.Validate(); err != nil { p.Clips[i].TransitionOut = old; return err }
 return nil
}

// normalizeTransitions limits existing overlaps after trims, splits or removals.
// Earlier overlaps take priority; each clip retains at least MinClipLength
// of non-overlapping footage, so FFmpeg always receives a positive segment.
func (p *Project) normalizeTransitions() {
 var incoming float64
 for i := range p.Clips {
  if i+1 >= len(p.Clips) { p.Clips[i].TransitionOut = Transition{}; break }
  trans := &p.Clips[i].TransitionOut
  if trans.Kind != "dissolve" && trans.Kind != "black" || !trans.Active() { *trans=Transition{}; incoming=0; continue }
  maxOut:=math.Min(p.Clips[i].Duration()-MinClipLength-incoming,p.Clips[i+1].Duration()-MinClipLength)
  if maxOut < 0.1 { *trans=Transition{}; incoming=0; continue }
  trans.Duration=math.Min(trans.Duration,maxOut)
  incoming=trans.Duration
 }
}
