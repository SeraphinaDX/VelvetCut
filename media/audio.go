package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/SeraphinaDX/VelvetCut/editor"
)

// ProbeAudio accepts WAV, MP3, FLAC, Ogg, AAC, M4A and audio tracks in
// video files; unlike Probe it does not require a video stream.
func ProbeAudio(ctx context.Context, path string) (float64, error) {
	var data struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			Duration  string `json:"duration"`
		} `json:"streams"`
	}
	raw, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration:stream=codec_type,duration", "-of", "json", path).Output()
	if err != nil {
		return 0, fmt.Errorf("probe audio %s: %w", filepath.Base(path), err)
	}
	if err = json.Unmarshal(raw, &data); err != nil {
		return 0, err
	}
	length, _ := strconv.ParseFloat(data.Format.Duration, 64)
	hasAudio := false
	for _, stream := range data.Streams {
		if stream.CodecType == "audio" {
			hasAudio = true
			if length <= 0 {
				length, _ = strconv.ParseFloat(stream.Duration, 64)
			}
		}
	}
	if !hasAudio || length < editor.MinClipLength {
		return 0, errors.New("file has no usable audio stream or duration")
	}
	return length, nil
}
