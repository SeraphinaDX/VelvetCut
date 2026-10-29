package editor

import "math"

// VisualSettings contains optional, non-destructive adjustments. Zero values
// intentionally reproduce the 0.5.x editor, so existing projects load cleanly.
// Contrast and saturation are deviations from 1, not absolute values.
type VisualSettings struct {
	TitleFont      string  `json:"title_font,omitempty"`      // Sans, Serif, Mono
	TitleColor     string  `json:"title_color,omitempty"`     // white, yellow, cyan, pink, black
	TitlePosition  string  `json:"title_position,omitempty"`  // bottom, top, center, lower-left
	TitleAnimation string  `json:"title_animation,omitempty"` // none, fade, slide, zoom
	TitleSize      int     `json:"title_size,omitempty"`      // 24..120, default 58 at 1280px wide
	TitleSeconds   float64 `json:"title_seconds,omitempty"`   // default 3s
	Brightness     float64 `json:"brightness,omitempty"`      // -0.5..0.5
	Contrast       float64 `json:"contrast,omitempty"`        // -0.5..1.0, add to 1
	Saturation     float64 `json:"saturation,omitempty"`      // -1..2, add to 1
	Temperature    float64 `json:"temperature,omitempty"`     // -1..1
	Effect         string  `json:"effect,omitempty"`          // none, bw, sepia, vignette
	Crop           string  `json:"crop,omitempty"`            // fit, fill, square, 4:3
	Rotation       int     `json:"rotation,omitempty"`        // 0, 90, 180, 270 clockwise
	Zoom           float64 `json:"zoom,omitempty"`            // 0 or 1..2.5
}

func (v VisualSettings) Valid() bool {
	validString := func(value string, allowed ...string) bool {
		if value == "" {
			return true
		}
		for _, candidate := range allowed {
			if value == candidate {
				return true
			}
		}
		return false
	}
	inRange := func(n, low, high float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) && n >= low && n <= high }
	return validString(v.TitleFont, "Sans", "Serif", "Mono") &&
		validString(v.TitleColor, "white", "yellow", "cyan", "pink", "black") &&
		validString(v.TitlePosition, "bottom", "top", "center", "lower-left") &&
		validString(v.TitleAnimation, "none", "fade", "slide", "zoom") &&
		(v.TitleSize == 0 || v.TitleSize >= 24 && v.TitleSize <= 120) &&
		inRange(v.TitleSeconds, 0, 30) &&
		inRange(v.Brightness, -0.5, 0.5) &&
		inRange(v.Contrast, -0.5, 1) &&
		inRange(v.Saturation, -1, 2) &&
		inRange(v.Temperature, -1, 1) &&
		validString(v.Effect, "none", "bw", "sepia", "vignette") &&
		validString(v.Crop, "fit", "fill", "square", "4:3") &&
		(v.Rotation == 0 || v.Rotation == 90 || v.Rotation == 180 || v.Rotation == 270) &&
		(v.Zoom == 0 || inRange(v.Zoom, 1, 2.5))
}

func (v VisualSettings) Active() bool { return v != (VisualSettings{}) }
func (v VisualSettings) TitleDuration(clipDuration float64) float64 {
	seconds := v.TitleSeconds
	if seconds == 0 {
		seconds = 3
	}
	return math.Min(seconds, clipDuration)
}
func (v VisualSettings) Size() int {
	if v.TitleSize == 0 {
		return 58
	}
	return v.TitleSize
}
func (v VisualSettings) CropZoom() float64 {
	if v.Zoom == 0 {
		return 1
	}
	return v.Zoom
}
