package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/SeraphinaDX/VelvetCut/editor"
)

// ClipInspector is an iMovie-style, docked panel. Controls are deliberately
// staged until Apply: every adjustment is one undo step and one proxy rebuild.
// Preview stills then reflect the same FFmpeg graph used for final export.
type ClipInspector struct {
	studio                                        *Studio
	content                                       fyne.CanvasObject
	selectedID                                    string
	loadedVisual                                  editor.VisualSettings
	loadedTitle                                   string
	header                                        *widget.Label
	applyButton                                   *widget.Button
	title                                         *widget.Entry
	font, color, position, animation              *widget.Select
	titleSize, titleSeconds                       *widget.Slider
	brightness, contrast, saturation, temperature *widget.Slider
	effect, crop, rotation                        *widget.Select
	zoom                                          *widget.Slider
	motionScale, motionOpacity                    *widget.Slider
	motionStatus                                  *widget.Label
	motionBusy                                    bool
}

func inspectorSelect(options []string) *widget.Select {
	s := widget.NewSelect(options, nil)
	if len(options) > 0 {
		s.SetSelected(options[0])
	}
	return s
}

func inspectorSlider(min, max, step float64) *widget.Slider {
	s := widget.NewSlider(min, max)
	s.Step = step
	return s
}

func labeledSlider(label string, slider *widget.Slider, format string) fyne.CanvasObject {
	value := widget.NewLabel(fmt.Sprintf(format, slider.Value))
	slider.OnChanged = func(v float64) { value.SetText(fmt.Sprintf(format, v)) }
	return container.NewVBox(container.NewBorder(nil, nil, widget.NewLabel(label), value), slider)
}

func newClipInspector(studio *Studio) *ClipInspector {
	in := &ClipInspector{studio: studio}
	in.header = widget.NewLabel("Select a video clip")
	in.title = widget.NewEntry()
	in.title.SetPlaceHolder("Title on the video")
	in.font = inspectorSelect([]string{"Sans", "Serif", "Mono"})
	in.color = inspectorSelect([]string{"white", "yellow", "cyan", "pink", "black"})
	in.position = inspectorSelect([]string{"bottom", "top", "center", "lower-left"})
	in.animation = inspectorSelect([]string{"none", "fade", "slide", "zoom"})
	in.titleSize = inspectorSlider(24, 120, 2)
	in.titleSeconds = inspectorSlider(0.5, 8, 0.5)
	in.brightness = inspectorSlider(-0.5, 0.5, 0.05)
	in.contrast = inspectorSlider(-0.5, 1, 0.05)
	in.saturation = inspectorSlider(-1, 2, 0.05)
	in.temperature = inspectorSlider(-1, 1, 0.05)
	in.effect = inspectorSelect([]string{"none", "bw", "sepia", "vignette"})
	in.crop = inspectorSelect([]string{"fit", "fill", "square", "4:3"})
	in.rotation = inspectorSelect([]string{"0", "90", "180", "270"})
	in.zoom = inspectorSlider(1, 2.5, 0.05)
	titles := container.NewVScroll(container.NewVBox(
		widget.NewLabel("On-screen title"), in.title,
		widget.NewForm(
			widget.NewFormItem("Font", in.font), widget.NewFormItem("Colour", in.color),
			widget.NewFormItem("Position", in.position), widget.NewFormItem("Animation", in.animation)),
		labeledSlider("Font size", in.titleSize, "%.0f px"),
		labeledSlider("Title duration", in.titleSeconds, "%.1f s"),
		widget.NewLabel("Title changes appear in the paused preview and the rendered movie."),
	))
	colour := container.NewVScroll(container.NewVBox(
		labeledSlider("Brightness", in.brightness, "%+.2f"),
		labeledSlider("Contrast", in.contrast, "%+.2f"),
		labeledSlider("Saturation", in.saturation, "%+.2f"),
		labeledSlider("Temperature (cool / warm)", in.temperature, "%+.2f"),
	))
	effects := container.NewVScroll(container.NewVBox(
		widget.NewForm(widget.NewFormItem("Effect", in.effect), widget.NewFormItem("Crop", in.crop), widget.NewFormItem("Rotate clockwise", in.rotation)),
		labeledSlider("Center crop zoom", in.zoom, "%.2f ×"),
		widget.NewLabel("Fit preserves the full image. Fill crops to the movie's aspect ratio. Square and 4:3 crop the source, then letterbox it."),
	))
	in.motionScale = inspectorSlider(0.25, 4, 0.05)
	in.motionOpacity = inspectorSlider(0, 1, 0.05)
	in.motionStatus = widget.NewLabel("Select a titled video clip to animate")
	in.motionStatus.Wrapping = fyne.TextWrapWord
	motion := container.NewVScroll(container.NewVBox(
		widget.NewLabel("Drag the title guide in the preview to create a position keyframe."),
		widget.NewLabel("At the playhead, set scale and opacity; click Add / update keyframe."),
		in.motionStatus,
		labeledSlider("Title size multiplier", in.motionScale, "%.2f ×"),
		labeledSlider("Title opacity", in.motionOpacity, "%.0f%%"),
		widget.NewButton("Add / update keyframe", in.putMotion),
		container.NewGridWithColumns(2,
			widget.NewButton("Previous keyframe", func() { in.jumpMotion(-1) }),
			widget.NewButton("Next keyframe", func() { in.jumpMotion(1) }),
			widget.NewButton("Remove at playhead", in.removeMotion),
			widget.NewButton("Clear all keyframes", in.clearMotion)),
		widget.NewLabel("Keyframes use clip-local time and interpolate linearly. They override legacy title entrance animations."),
	))
	tabs := container.NewAppTabs(container.NewTabItem("Title", titles), container.NewTabItem("Colour", colour), container.NewTabItem("Effects", effects), container.NewTabItem("Motion", motion))
	tabs.SetTabLocation(container.TabLocationTop)
	in.applyButton = widget.NewButton("Apply to selected clip", in.apply)
	previewButton := widget.NewButton("Show clip", func() {
		if idx := studio.selected; idx >= 0 && idx < len(studio.project.Clips) {
			studio.pause()
			studio.seek(studio.project.Start(idx) + math.Min(0.4, studio.project.Clips[idx].Duration()/2))
		}
	})
	in.content = container.NewBorder(
		container.NewVBox(widget.NewLabelWithStyle("CLIP INSPECTOR", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), in.header),
		container.NewVBox(in.applyButton, previewButton), nil, nil, tabs)
	in.applyButton.Disable()
	return in
}

func (in *ClipInspector) loadSelection() {
	s := in.studio
	if s.selected < 0 || s.selected >= len(s.project.Clips) || s.selectedAudio >= 0 {
		in.selectedID = ""
		in.header.SetText("Select a video clip")
		in.applyButton.Disable()
		in.refreshMotion()
		return
	}
	c := s.project.Clips[s.selected]
	if c.ID == in.selectedID && c.Visual == in.loadedVisual && c.Title == in.loadedTitle {
		in.refreshMotion()
		return
	}
	in.loadedVisual = c.Visual
	in.loadedTitle = c.Title
	in.selectedID = c.ID
	in.header.SetText(c.Name)
	in.applyButton.Enable()
	v := c.Visual
	in.title.SetText(c.Title)
	set := func(s *widget.Select, value, def string) {
		if value == "" {
			value = def
		}
		s.SetSelected(value)
	}
	set(in.font, v.TitleFont, "Sans")
	set(in.color, v.TitleColor, "white")
	set(in.position, v.TitlePosition, "bottom")
	set(in.animation, v.TitleAnimation, "none")
	size := float64(v.Size())
	in.titleSize.SetValue(size)
	seconds := v.TitleSeconds
	if seconds == 0 {
		seconds = 3
	}
	in.titleSeconds.SetValue(seconds)
	in.brightness.SetValue(v.Brightness)
	in.contrast.SetValue(v.Contrast)
	in.saturation.SetValue(v.Saturation)
	in.temperature.SetValue(v.Temperature)
	set(in.effect, v.Effect, "none")
	set(in.crop, v.Crop, "fit")
	in.rotation.SetSelected(strconv.Itoa(v.Rotation))
	in.zoom.SetValue(v.CropZoom())
	in.refreshMotion()
}

func (in *ClipInspector) apply() {
	s := in.studio
	idx := s.selected
	if idx < 0 || idx >= len(s.project.Clips) || s.project.Clips[idx].ID != in.selectedID {
		return
	}
	original := s.project.Clips[idx]
	changed := original
	changed.Title = strings.TrimSpace(in.title.Text)
	v := editor.VisualSettings{
		TitleFont: in.font.Selected, TitleColor: in.color.Selected, TitlePosition: in.position.Selected,
		TitleAnimation: in.animation.Selected, TitleSize: int(math.Round(in.titleSize.Value)),
		TitleSeconds: in.titleSeconds.Value,
		Brightness:   in.brightness.Value, Contrast: in.contrast.Value, Saturation: in.saturation.Value,
		Temperature: in.temperature.Value, Effect: in.effect.Selected, Crop: in.crop.Selected,
		Zoom: in.zoom.Value,
	}
	v.Rotation, _ = strconv.Atoi(in.rotation.Selected)
	// Keep zero-value fields lean and compatible with pre-0.6 projects.
	if v.TitleFont == "Sans" {
		v.TitleFont = ""
	}
	if v.TitleColor == "white" {
		v.TitleColor = ""
	}
	if v.TitlePosition == "bottom" {
		v.TitlePosition = ""
	}
	if v.TitleAnimation == "none" {
		v.TitleAnimation = ""
	}
	if v.TitleSize == 58 {
		v.TitleSize = 0
	}
	if v.TitleSeconds == 3 {
		v.TitleSeconds = 0
	}
	if v.Effect == "none" {
		v.Effect = ""
	}
	if v.Crop == "fit" {
		v.Crop = ""
	}
	if math.Abs(v.Zoom-1) < 0.001 {
		v.Zoom = 0
	}
	changed.Visual = v
	if !changed.Valid() {
		s.problem(fmt.Errorf("clip visual settings are invalid"))
		return
	}
	if original.Title == changed.Title && original.Visual == changed.Visual {
		s.info.SetText("No inspector changes to apply")
		return
	}
	s.pause()
	s.remember()
	s.project.Clips[idx] = changed
	s.changed("Applied visual settings · preview and export use the same effects")
	s.seek(s.project.Start(idx) + math.Min(0.4, changed.Duration()/2))
}

// refreshMotion only reloads values when the playhead moves; dragging the
// inspector sliders never mutates the project until Add/Update is pressed.
func (in *ClipInspector) refreshMotion() {
	if in.motionStatus == nil || in.motionBusy {
		return
	}
	s := in.studio
	if s.selectedAudio >= 0 || s.selected < 0 || s.selected >= len(s.project.Clips) {
		in.motionStatus.SetText("Select a video clip")
		return
	}
	clip := s.project.Clips[s.selected]
	local := s.titleKeyframeTime(s.selected)
	pose := clip.InitialTitleKeyframe(local)
	in.motionBusy = true
	in.motionScale.SetValue(pose.Scale)
	in.motionOpacity.SetValue(pose.Opacity)
	in.motionBusy = false
	in.motionStatus.SetText(fmt.Sprintf("Clip %s / %s · %d keyframe(s) · position %.0f%%, %.0f%%", timecode(local), timecode(clip.Duration()), len(clip.TitleKeyframes), pose.X*100, pose.Y*100))
}

func (in *ClipInspector) putMotion() {
	s := in.studio
	i := s.selected
	if s.selectedAudio >= 0 || i < 0 || i >= len(s.project.Clips) {
		return
	}
	clip := &s.project.Clips[i]
	if strings.TrimSpace(clip.Title) == "" {
		s.info.SetText("Enter a title and Apply first, then add motion")
		return
	}
	s.pause()
	local := s.titleKeyframeTime(i)
	pose := clip.InitialTitleKeyframe(local)
	pose.Time = local
	pose.Scale = math.Round(in.motionScale.Value*100) / 100
	pose.Opacity = math.Round(in.motionOpacity.Value*100) / 100
	s.remember()
	if !clip.PutTitleKeyframe(pose) {
		s.undo = s.undo[:len(s.undo)-1]
		return
	}
	s.changed("Saved title keyframe at " + timecode(local))
	s.seek(s.playhead)
}

func (in *ClipInspector) removeMotion() {
	s := in.studio
	i := s.selected
	if i < 0 || i >= len(s.project.Clips) || s.selectedAudio >= 0 {
		return
	}
	local := s.titleKeyframeTime(i)
	clip := &s.project.Clips[i]
	found := false
	for _, key := range clip.TitleKeyframes {
		if math.Abs(key.Time-local) <= 1.0/60 {
			found = true
			break
		}
	}
	if !found {
		s.info.SetText("Move the playhead onto a keyframe to delete it")
		return
	}
	s.pause()
	s.remember()
	clip.DeleteTitleKeyframe(local)
	s.changed("Removed title keyframe")
	s.seek(s.playhead)
}

func (in *ClipInspector) clearMotion() {
	s := in.studio
	i := s.selected
	if i < 0 || i >= len(s.project.Clips) || s.selectedAudio >= 0 || len(s.project.Clips[i].TitleKeyframes) == 0 {
		return
	}
	s.pause()
	s.remember()
	s.project.Clips[i].TitleKeyframes = nil
	s.changed("Cleared title animation")
	s.seek(s.playhead)
}

func (in *ClipInspector) jumpMotion(direction int) {
	s := in.studio
	i := s.selected
	if i < 0 || i >= len(s.project.Clips) || s.selectedAudio >= 0 {
		return
	}
	local := s.titleKeyframeTime(i)
	keys := s.project.Clips[i].TitleKeyframes
	var target float64
	found := false
	if direction > 0 {
		for _, k := range keys {
			if k.Time > local+1.0/60 {
				target = k.Time
				found = true
				break
			}
		}
	} else {
		for j := len(keys) - 1; j >= 0; j-- {
			if keys[j].Time < local-1.0/60 {
				target = keys[j].Time
				found = true
				break
			}
		}
	}
	if found {
		s.pause()
		s.seek(s.project.Start(i) + target)
	}
}
