package media

import (
	"bufio"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"os/exec"
)

// PreviewFrame is a low-resolution still from a continuously decoded video
// stream. The mpv audio clock (not the FFmpeg decode rate) determines display.
type PreviewFrame struct {
	Image      image.Image
	SourceTime float64
}

// StreamFrames seeks once and incrementally decodes sequential JPEGs. The small
// bounded channel back-pressures FFmpeg rather than buffering an entire movie.
// Cancel ctx when the user seeks, switches clips, or stops playback.
func StreamFrames(ctx context.Context, path string, from, duration float64, fps int) (<-chan PreviewFrame, error) {
	if path == "" || math.IsNaN(from) || math.IsInf(from, 0) || from < 0 || duration <= 0 || math.IsNaN(duration) || math.IsInf(duration, 0) || fps < 1 || fps > 30 {
		return nil, fmt.Errorf("invalid video preview range")
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-ss", seconds(from), "-t", seconds(duration), "-i", path,
		"-an", "-vf", fmt.Sprintf("fps=%d,scale=640:360:force_original_aspect_ratio=decrease", fps), "-q:v", "5", "-f", "image2pipe", "-c:v", "mjpeg", "pipe:1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	frames := make(chan PreviewFrame, 6)
	go func() {
		defer close(frames)
		defer func() { _ = stdout.Close(); _ = cmd.Wait() }()
		reader := bufio.NewReaderSize(stdout, 32*1024)
		for index := 0; ; index++ {
			frame, err := jpeg.Decode(reader)
			if err != nil {
				return
			}
			packet := PreviewFrame{Image: frame, SourceTime: from + float64(index)/float64(fps)}
			select {
			case frames <- packet:
			case <-ctx.Done():
				return
			}
		}
	}()
	return frames, nil
}
