package editor

import (
	"math"
	"sort"
)

// TitleKeyframe is a title transform at a time relative to the beginning of
// a trimmed clip (not the source file). X and Y are normalized centre anchors
// in the rendered movie. Scale and Opacity are explicit, so old projects with
// no keyframes keep their previous title behaviour.
type TitleKeyframe struct {
	Time    float64 `json:"time"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Scale   float64 `json:"scale"`
	Opacity float64 `json:"opacity"`
}

func (k TitleKeyframe) Valid() bool {
	return isFinite(k.Time) && k.Time >= 0 &&
		isFinite(k.X) && k.X >= 0 && k.X <= 1 &&
		isFinite(k.Y) && k.Y >= 0 && k.Y <= 1 &&
		isFinite(k.Scale) && k.Scale >= 0.25 && k.Scale <= 4 &&
		isFinite(k.Opacity) && k.Opacity >= 0 && k.Opacity <= 1
}

func titleKeyframesValid(keys []TitleKeyframe, duration float64) bool {
	last := -1.0
	for _, k := range keys {
		if !k.Valid() || k.Time > duration+1e-5 || k.Time <= last+1e-6 {
			return false
		}
		last = k.Time
	}
	return true
}

// TitleAt linearly interpolates position, size and opacity between keyframes.
// Values hold before the first and after the last keyframe.
func (c Clip) TitleAt(time float64) (TitleKeyframe, bool) {
	keys := c.TitleKeyframes
	if len(keys) == 0 {
		return TitleKeyframe{}, false
	}
	if time <= keys[0].Time {
		return keys[0], true
	}
	for i := 1; i < len(keys); i++ {
		if time <= keys[i].Time {
			before, after := keys[i-1], keys[i]
			fraction := (time - before.Time) / (after.Time - before.Time)
			return TitleKeyframe{
				Time:    time,
				X:       before.X + fraction*(after.X-before.X),
				Y:       before.Y + fraction*(after.Y-before.Y),
				Scale:   before.Scale + fraction*(after.Scale-before.Scale),
				Opacity: before.Opacity + fraction*(after.Opacity-before.Opacity),
			}, true
		}
	}
	return keys[len(keys)-1], true
}

// InitialTitleKeyframe translates legacy presets into a screen-centred anchor.
func (c Clip) InitialTitleKeyframe(time float64) TitleKeyframe {
	if pose, ok := c.TitleAt(time); ok {
		pose.Time = time
		return pose
	}
	result := TitleKeyframe{Time: time, X: 0.5, Y: 0.80, Scale: 1, Opacity: 1}
	switch c.Visual.TitlePosition {
	case "top":
		result.Y = 0.14
	case "center":
		result.Y = 0.5
	case "lower-left":
		result.X = 0.17
	}
	return result
}

// PutTitleKeyframe adds or replaces a keyframe without changing the source.
// Times within half a 30-fps frame replace one another, preventing near-duplicate
// keyframes from mouse drags and repeatedly pressing Add.
func (c *Clip) PutTitleKeyframe(k TitleKeyframe) bool {
	if !k.Valid() || k.Time > c.Duration()+1e-5 {
		return false
	}
	for i, old := range c.TitleKeyframes {
		if math.Abs(old.Time-k.Time) <= 1.0/60 {
			k.Time = old.Time
			c.TitleKeyframes[i] = k
			return true
		}
	}
	c.TitleKeyframes = append(c.TitleKeyframes, k)
	sort.Slice(c.TitleKeyframes, func(i, j int) bool { return c.TitleKeyframes[i].Time < c.TitleKeyframes[j].Time })
	return true
}

func (c *Clip) DeleteTitleKeyframe(time float64) bool {
	for i, k := range c.TitleKeyframes {
		if math.Abs(k.Time-time) <= 1.0/60 {
			c.TitleKeyframes = append(c.TitleKeyframes[:i], c.TitleKeyframes[i+1:]...)
			return true
		}
	}
	return false
}

// titleKeyframeRange remaps an original clip's motion onto the retained
// interval. The interpolated boundary pose keeps animation continuous when
// splitting or trimming a clip. A negative start permits extending a trim.
func titleKeyframeRange(keys []TitleKeyframe, start, end float64) []TitleKeyframe {
	if len(keys) == 0 || end <= start {
		return nil
	}
	clip := Clip{TitleKeyframes: keys}
	out := make([]TitleKeyframe, 0, len(keys)+2)
	left, _ := clip.TitleAt(start)
	left.Time = 0
	out = append(out, left)
	for _, key := range keys {
		if key.Time > start+1e-6 && key.Time < end-1e-6 {
			key.Time -= start
			out = append(out, key)
		}
	}
	right, _ := clip.TitleAt(end)
	right.Time = end - start
	if right.Time > 1e-6 {
		out = append(out, right)
	}
	return out
}
