package media

import (
	"context"
	"encoding/binary"
	"math"
	"os/exec"
)

// Waveform returns peaks evenly spaced over a source file's audio stream.
// Rendering a peak per few horizontal pixels avoids drawing millions of samples.
func Waveform(ctx context.Context, path string, duration float64) ([]float32, error) {
	if duration <= 0 {
		return nil, nil
	}
	bins := int(math.Min(3000, math.Max(100, duration*15)))
	// Low sample rate suffices for amplitude overview, not for audio playback.
	data, err := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-i", path, "-map", "0:a:0", "-vn", "-ac", "1", "-ar", "200", "-f", "s16le", "pipe:1").Output()
	if err != nil {
		return nil, err
	}
	n := len(data) / 2
	if n == 0 {
		return make([]float32, bins), nil
	}
	peaks := make([]float32, bins)
	for j := 0; j < n; j++ {
		v := int16(binary.LittleEndian.Uint16(data[j*2:]))
		abs := float32(math.Abs(float64(v)) / 32768)
		bin := int(float64(j) * float64(bins) / float64(n))
		if bin >= bins {
			bin = bins - 1
		}
		if abs > peaks[bin] {
			peaks[bin] = abs
		}
	}
	return peaks, nil
}
