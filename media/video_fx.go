package media

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/SeraphinaDX/VelvetCut/editor"
)

// visualFilters is shared by full export, preview proxy and paused stills.
// All text values except the textfile path come from validated enums, avoiding
// FFmpeg expression injection. File contents are never interpreted as filters.
func visualFilters(clip editor.Clip, width, height, fps int, titleFile string, stillAt float64) string {
	v := clip.Visual
	duration := clip.Duration()
	pts := "setpts=PTS-STARTPTS"
	if stillAt >= 0 {
		pts += "+" + seconds(stillAt) + "/TB"
	}
	parts := []string{pts, fmt.Sprintf("fps=%d", fps)}
	switch v.Rotation {
	case 90:
		parts = append(parts, "transpose=clock")
	case 180:
		parts = append(parts, "hflip,vflip")
	case 270:
		parts = append(parts, "transpose=cclock")
	}
	ratio := ""
	switch v.Crop {
	case "fill":
		ratio = fmt.Sprintf("%d/%d", width, height)
	case "square":
		ratio = "1"
	case "4:3":
		ratio = "4/3"
	}
	if ratio != "" {
		parts = append(parts, fmt.Sprintf("crop=w='min(iw,ih*(%s))':h='min(ih,iw/(%s))':x=(iw-ow)/2:y=(ih-oh)/2", ratio, ratio))
	}
	if zoom := v.CropZoom(); zoom > 1.001 {
		parts = append(parts, fmt.Sprintf("crop=w=iw/%s:h=ih/%s:x=(iw-ow)/2:y=(ih-oh)/2", seconds(zoom), seconds(zoom)))
	}
	parts = append(parts, fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1", width, height, width, height))
	if len(clip.VideoKeyframes) > 0 {
		// Stabilize the canvas first, then animate its crop. zoompan evaluates
		// per incoming video frame. The input timestamps are reset at the
		// beginning of this filter chain; `it` is clip-local seconds.
		// Setting d=1 ensures one output frame per incoming frame rather
		// than accidentally creating 25 frames per input frame.
		zoom := videoKeyframeExpression(clip.VideoKeyframes, func(k editor.VideoKeyframe) float64 { return k.Zoom })
		x := videoKeyframeExpression(clip.VideoKeyframes, func(k editor.VideoKeyframe) float64 { return k.X })
		y := videoKeyframeExpression(clip.VideoKeyframes, func(k editor.VideoKeyframe) float64 { return k.Y })
		parts = append(parts, fmt.Sprintf("zoompan=z='%s':x='(iw-iw/zoom)*(%s)':y='(ih-ih/zoom)*(%s)':d=1:s=%dx%d:fps=%d", zoom, x, y, width, height, fps))
	}
	if v.Brightness != 0 || v.Contrast != 0 || v.Saturation != 0 {
		parts = append(parts, fmt.Sprintf("eq=brightness=%s:contrast=%s:saturation=%s", seconds(v.Brightness), seconds(1+v.Contrast), seconds(1+v.Saturation)))
	}
	if v.Temperature != 0 {
		// Red/blue tint; the midpoint leaves the image unchanged.
		t := v.Temperature * 0.30
		parts = append(parts, fmt.Sprintf("colorbalance=rs=%s:gs=0:bs=%s:rm=%s:gm=0:bm=%s", seconds(t), seconds(-t), seconds(t), seconds(-t)))
	}
	switch v.Effect {
	case "bw":
		parts = append(parts, "hue=s=0")
	case "sepia":
		parts = append(parts, "colorchannelmixer=rr=.393:rg=.769:rb=.189:gr=.349:gg=.686:gb=.168:br=.272:bg=.534:bb=.131")
	case "vignette":
		parts = append(parts, "vignette=angle=PI/5")
	}
	if clip.FadeIn > 0 {
		parts = append(parts, "fade=t=in:st=0:d="+seconds(clip.FadeIn))
	}
	if clip.FadeOut > 0 {
		parts = append(parts, "fade=t=out:st="+seconds(duration-clip.FadeOut)+":d="+seconds(clip.FadeOut))
	}
	if clip.Title != "" && titleFile != "" {
		font := v.TitleFont
		if font == "" {
			font = "Sans"
		}
		if font == "Mono" {
			font = "Monospace"
		}
		color := v.TitleColor
		if color == "" {
			color = "white"
		}
		switch color {
		case "pink":
			color = "0xFFC0F3"
		}
		size := float64(v.Size()) * float64(width) / 1280
		size = math.Max(10, size)
		x, y := "(w-text_w)/2", "h*0.76"
		switch v.TitlePosition {
		case "top":
			y = "h*0.10"
		case "center":
			y = "(h-text_h)/2"
		case "lower-left":
			x = "w*0.07"
			y = "h*0.76"
		}
		limit := v.TitleDuration(duration)
		alpha := ""
		switch v.TitleAnimation {
		case "fade":
			alpha = fmt.Sprintf(":alpha='min(1\\,t/0.4)*min(1\\,(%s-t)/0.4)'", seconds(limit))
		case "slide":
			x = fmt.Sprintf("if(lt(t\\,0.5)\\,-text_w+(%s+text_w)*t/0.5\\,%s)", x, x)
		case "zoom":
			if len(clip.TitleKeyframes) > 0 {
				break
			}
			// drawtext evaluates fontsize expressions at each frame.
			sizeExpr := fmt.Sprintf("'%s*(0.65+0.35*min(t/0.4\\,1))'", seconds(size))
			alpha = ""
			parts = append(parts, fmt.Sprintf("drawtext=font=%s:textfile=%s:expansion=none:fontcolor=%s:fontsize=%s:box=1:boxcolor=black@0.5:boxborderw=12:x='%s':y='%s':enable='lt(t,%s)'", font, titleFile, color, sizeExpr, x, y, seconds(limit)))
		}
		if len(clip.TitleKeyframes) > 0 {
			// Keyframed motion overrides the old preset animation. Drawtext
			// evaluates these expressions at each video frame, so export,
			// effects proxy and paused stills all use the same interpolation.
			x = "w*(" + keyframeExpression(clip.TitleKeyframes, func(k editor.TitleKeyframe) float64 { return k.X }) + ")-text_w/2"
			y = "h*(" + keyframeExpression(clip.TitleKeyframes, func(k editor.TitleKeyframe) float64 { return k.Y }) + ")-text_h/2"
			scale := keyframeExpression(clip.TitleKeyframes, func(k editor.TitleKeyframe) float64 { return k.Scale })
			opacity := keyframeExpression(clip.TitleKeyframes, func(k editor.TitleKeyframe) float64 { return k.Opacity })
			parts = append(parts, fmt.Sprintf("drawtext=font=%s:textfile=%s:expansion=none:fontcolor=%s:fontsize='%s*(%s)':alpha='%s':box=1:boxcolor=black@0.5:boxborderw=12:x='%s':y='%s':enable='lt(t,%s)'", font, titleFile, color, seconds(size), scale, opacity, x, y, seconds(limit)))
		} else if v.TitleAnimation != "zoom" {
			parts = append(parts, fmt.Sprintf("drawtext=font=%s:textfile=%s:expansion=none:fontcolor=%s:fontsize=%s:box=1:boxcolor=black@0.5:boxborderw=12:x='%s':y='%s'%s:enable='lt(t,%s)'", font, titleFile, color, seconds(size), x, y, alpha, seconds(limit)))
		}
	}
	parts = append(parts, fmt.Sprintf("format=yuv444p,settb=1/%d", fps))
	return strings.Join(parts, ",")
}

// keyframeExpression emits one piecewise-linear FFmpeg expression. Escaped
// commas are required because drawtext is inside a filter_complex graph.
// Times are clip-local, and values hold outside the keyframe range.
func keyframeExpression(keys []editor.TitleKeyframe, value func(editor.TitleKeyframe) float64) string {
	if len(keys) == 0 {
		return "0"
	}
	expr := seconds(value(keys[len(keys)-1]))
	for i := len(keys) - 2; i >= 0; i-- {
		a, b := keys[i], keys[i+1]
		span := b.Time - a.Time
		interp := fmt.Sprintf("(%s+(%s-%s)*(t-%s)/%s)", seconds(value(a)), seconds(value(b)), seconds(value(a)), seconds(a.Time), seconds(span))
		expr = fmt.Sprintf("if(lt(t\\,%s)\\,%s\\,if(lt(t\\,%s)\\,%s\\,%s))", seconds(a.Time), seconds(value(a)), seconds(b.Time), interp, expr)
	}
	return expr
}

// videoKeyframeExpression uses zoompan's input timestamp (`it`). Each
// segment is linear, holding values before the first and after the last key.
// Escaped commas prevent FFmpeg from splitting the nested expression.
func videoKeyframeExpression(keys []editor.VideoKeyframe, value func(editor.VideoKeyframe) float64) string {
	if len(keys) == 0 {
		return "0"
	}
	expr := seconds(value(keys[len(keys)-1]))
	for i := len(keys) - 2; i >= 0; i-- {
		a, b := keys[i], keys[i+1]
		interp := fmt.Sprintf("(%s+(%s-%s)*(it-%s)/%s)", seconds(value(a)), seconds(value(b)), seconds(value(a)), seconds(a.Time), seconds(b.Time-a.Time))
		expr = fmt.Sprintf("if(lt(it\\,%s)\\,%s\\,if(lt(it\\,%s)\\,%s\\,%s))", seconds(a.Time), seconds(value(a)), seconds(b.Time), interp, expr)
	}
	return expr
}

func makeTitleFile(dir string, index int, text string) (string, error) {
	if text == "" {
		return "", nil
	}
	path := filepath.Join(dir, fmt.Sprintf("title-%d.txt", index))
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		return "", err
	}
	return path, nil
}

// StyledFrame renders a single time-aligned frame using the exact same visual
// filters as MP4 export. It is used while paused/scrubbing, before a full
// playback proxy has been produced, so inspectors visibly affect the preview.
func StyledFrame(ctx context.Context, clip editor.Clip, sourceTime float64) (image.Image, error) {
	if !clip.Valid() {
		return nil, fmt.Errorf("invalid clip")
	}
	dir, err := os.MkdirTemp("", "velvetcut-frame-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	title, err := makeTitleFile(dir, 0, clip.Title)
	if err != nil {
		return nil, err
	}
	local := math.Max(0, math.Min(sourceTime-clip.SourceStart, clip.Duration()-0.001))
	vf := visualFilters(clip, 960, 540, 30, title, local)
	args := []string{"-hide_banner", "-loglevel", "error", "-ss", seconds(sourceTime), "-i", clip.Path, "-frames:v", "1", "-vf", vf, "-f", "image2pipe", "-vcodec", "mjpeg", "pipe:1"}
	data, err := exec.CommandContext(ctx, "ffmpeg", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("extract styled frame: %w", err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}
