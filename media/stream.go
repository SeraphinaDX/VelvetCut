package media

import (
	"context"
	"fmt"
	"image"
	"io"
	"math"
	"os/exec"
)

const (
	previewWidth  = 640
	previewHeight = 360
)

// PreviewFrame is a low-resolution frame from a continuously decoded video
// stream. mpv's playback position, not FFmpeg's decode speed, controls display.
type PreviewFrame struct {
	Image      image.Image
	SourceTime float64
}

// StreamFrames seeks once and decodes sequential *fixed-size* RGBA images.
// JPEG decoding directly from an image2pipe MJPEG stream is unsafe here:
// jpeg.Decode can read past one JPEG's EOI marker, silently swallowing the
// beginning of the next image. This made the embedded preview freeze after
// its first frame while mpv continued playing audio.
//
// Raw RGBA frames are unambiguously delimited by their byte length, and
// padding keeps the preview dimensions constant for portrait and 4:3 clips.
// The two-frame channel back-pressures FFmpeg, keeping memory bounded.
// Cancel ctx on seek, stop, or clip switch.
func StreamFrames(ctx context.Context, path string, from, duration float64, fps int) (<-chan PreviewFrame, error) {
	if path == "" || math.IsNaN(from) || math.IsInf(from, 0) || from < 0 || duration <= 0 || math.IsNaN(duration) || math.IsInf(duration, 0) || fps < 1 || fps > 30 {
		return nil, fmt.Errorf("invalid video preview range")
	}
	filter := fmt.Sprintf("fps=%d,scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,format=rgba",
		fps, previewWidth, previewHeight, previewWidth, previewHeight)
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-ss", seconds(from), "-t", seconds(duration), "-i", path,
		"-an", "-vf", filter, "-pix_fmt", "rgba", "-f", "rawvideo", "pipe:1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	frames := make(chan PreviewFrame, 2)
	go func() {
		defer close(frames)
		defer func() { _ = stdout.Close(); _ = cmd.Wait() }()
		for index := 0; ; index++ {
			img := image.NewRGBA(image.Rect(0, 0, previewWidth, previewHeight))
			if _, err := io.ReadFull(stdout, img.Pix); err != nil {
				return // clean EOF, cancellation, or a failed/incomplete FFmpeg stream
			}
			packet := PreviewFrame{Image: img, SourceTime: from + float64(index)/float64(fps)}
			select {
			case frames <- packet:
			case <-ctx.Done():
				return
			}
		}
	}()
	return frames, nil
}
