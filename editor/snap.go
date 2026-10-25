package editor

import "math"

// SnapTime returns the closest clip boundary within tolerance seconds.
// Includes the beginning and end of the movie and keeps free scrubbing possible
// when no boundary is nearby. Pure timeline time, independent of clip sources.
func (p Project) SnapTime(at, tolerance float64) float64 {
	if !isFinite(at) || !isFinite(tolerance) || tolerance <= 0 || len(p.Clips) == 0 {
		return at
	}
	closest, distance := at, tolerance
	cut := 0.0
	if d := math.Abs(at - cut); d <= distance {
		closest, distance = cut, d
	}
	for _, c := range p.Clips {
		cut += c.Duration()
		if d := math.Abs(at - cut); d <= distance {
			closest, distance = cut, d
		}
	}
	return closest
}
