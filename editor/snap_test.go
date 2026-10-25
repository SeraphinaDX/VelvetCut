package editor

import (
	"math"
	"testing"
)

func TestSnapTime(t *testing.T) {
	p := NewProject()
	for i, d := range []float64{2.5, 3.5, 4} {
		p.Clips = append(p.Clips, Clip{ID: string(rune('A' + i)), Path: "clip.mp4", Name: "clip", SourceStart: 0, SourceEnd: d, SourceDuration: d})
	}
	cases := []struct{ at, tol, want float64 }{
		{2.41, .1, 2.5}, {2.61, .1, 2.61}, {5.92, .1, 6}, {9.92, .1, 10}, {.06, .1, 0}, {1.3, .1, 1.3}, {2.4, 0, 2.4},
	}
	for _, c := range cases {
		got := p.SnapTime(c.at, c.tol)
		if math.Abs(got-c.want) > 1e-8 {
			t.Fatalf("SnapTime(%v,%v)=%v, expected %v", c.at, c.tol, got, c.want)
		}
	}
}
