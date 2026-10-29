package editor

import (
	"math"
	"sort"
)

// VideoKeyframe describes Ken Burns-style framing for a video clip.
// Time is relative to the trimmed clip; X/Y choose the normalized focus
// within the full frame (0=left/top, 1=right/bottom) and Zoom is 1..4.
// Without keyframes the original image is displayed unchanged.
type VideoKeyframe struct {
	Time float64 `json:"time"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

func (k VideoKeyframe) Valid() bool {
	return isFinite(k.Time) && k.Time >= 0 &&
		isFinite(k.X) && k.X >= 0 && k.X <= 1 &&
		isFinite(k.Y) && k.Y >= 0 && k.Y <= 1 &&
		isFinite(k.Zoom) && k.Zoom >= 1 && k.Zoom <= 4
}

func videoKeyframesValid(keys []VideoKeyframe, duration float64) bool {
	last := -1.0
	for _, k := range keys {
		if !k.Valid() || k.Time > duration+1e-5 || k.Time <= last+1e-6 {
			return false
		}
		last = k.Time
	}
	return true
}

// VideoAt interpolates between keyframes, holding the first/last poses outside
// their timestamps. It returns false for pre-0.8 projects without camera motion.
func (c Clip) VideoAt(time float64) (VideoKeyframe, bool) {
	keys := c.VideoKeyframes
	if len(keys) == 0 {
		return VideoKeyframe{}, false
	}
	if time <= keys[0].Time {
		return keys[0], true
	}
	for i := 1; i < len(keys); i++ {
		if time <= keys[i].Time {
			a, b := keys[i-1], keys[i]
			f := (time - a.Time) / (b.Time - a.Time)
			return VideoKeyframe{Time: time, X: a.X + f*(b.X-a.X), Y: a.Y + f*(b.Y-a.Y), Zoom: a.Zoom + f*(b.Zoom-a.Zoom)}, true
		}
	}
	return keys[len(keys)-1], true
}

func (c Clip) InitialVideoKeyframe(time float64) VideoKeyframe {
	if pose, ok := c.VideoAt(time); ok {
		pose.Time = time
		return pose
	}
	return VideoKeyframe{Time: time, X: 0.5, Y: 0.5, Zoom: 1}
}

func (c *Clip) PutVideoKeyframe(k VideoKeyframe) bool {
	if !k.Valid() || k.Time > c.Duration()+1e-5 {
		return false
	}
	for i, old := range c.VideoKeyframes {
		if math.Abs(old.Time-k.Time) <= 1.0/60 {
			k.Time = old.Time
			c.VideoKeyframes[i] = k
			return true
		}
	}
	c.VideoKeyframes = append(c.VideoKeyframes, k)
	sort.Slice(c.VideoKeyframes, func(i, j int) bool { return c.VideoKeyframes[i].Time < c.VideoKeyframes[j].Time })
	return true
}

func (c *Clip) DeleteVideoKeyframe(time float64) bool {
	for i, key := range c.VideoKeyframes {
		if math.Abs(key.Time-time) <= 1.0/60 {
			c.VideoKeyframes = append(c.VideoKeyframes[:i], c.VideoKeyframes[i+1:]...)
			return true
		}
	}
	return false
}

// Interpolated end poses ensure trims and splits preserve the camera movement.
func videoKeyframeRange(keys []VideoKeyframe, start, end float64) []VideoKeyframe {
	if len(keys) == 0 || end <= start {
		return nil
	}
	clip := Clip{VideoKeyframes: keys}
	out := make([]VideoKeyframe, 0, len(keys)+2)
	left, _ := clip.VideoAt(start)
	left.Time = 0
	out = append(out, left)
	for _, k := range keys {
		if k.Time > start+1e-6 && k.Time < end-1e-6 {
			k.Time -= start
			out = append(out, k)
		}
	}
	right, _ := clip.VideoAt(end)
	right.Time = end - start
	if right.Time > 1e-6 {
		out = append(out, right)
	}
	return out
}
