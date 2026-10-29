package main

import (
	"image/color"
	"math"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/SeraphinaDX/VelvetCut/editor"
)

// TitleOverlay draws a drag handle over the actual rendered preview title.
// The underlying preview image is still rendered by FFmpeg; keeping this
// overlay a translucent guide avoids drawing the title text twice.
// A drag commits ONE keyframe and ONE undo step when the mouse is released.
type TitleOverlay struct {
	widget.BaseWidget
	studio      *Studio
	root        *fyne.Container
	frame       *canvas.Rectangle
	hint        *canvas.Text
	pose        editor.TitleKeyframe
	cameraPose  editor.VideoKeyframe
	cameraDrag  bool
	moving      bool
	canDrag     bool
	frameX      float32
	frameY      float32
	frameWidth  float32
	frameHeight float32
}

func newTitleOverlay(s *Studio) *TitleOverlay {
	o := &TitleOverlay{studio: s}
	o.ExtendBaseWidget(o)
	return o
}

func (o *TitleOverlay) CreateRenderer() fyne.WidgetRenderer {
	o.frame = canvas.NewRectangle(color.NRGBA{R: 183, G: 138, B: 248, A: 24})
	o.frame.StrokeColor = color.NRGBA{R: 242, G: 224, B: 255, A: 245}
	o.frame.StrokeWidth = 1.5
	o.hint = canvas.NewText("↕ Drag title", color.NRGBA{R: 255, G: 239, B: 184, A: 255})
	o.hint.TextSize = 11
	o.root = container.NewWithoutLayout(o.frame, o.hint)
	o.layoutGuide()
	return widget.NewSimpleRenderer(o.root)
}

func (o *TitleOverlay) Resize(size fyne.Size) {
	o.BaseWidget.Resize(size)
	o.layoutGuide()
}

func (o *TitleOverlay) Refresh() {
	o.layoutGuide()
	o.BaseWidget.Refresh()
}

// videoRect maps normalized positions to the letterboxed 16:9 movie inside
// the responsive preview panel; no Fyne window pixel coordinates are stored.
func (o *TitleOverlay) videoRect() (x, y, width, height float32) {
	size := o.Size()
	width, height = size.Width, size.Height
	if width <= 0 || height <= 0 {
		return 0, 0, 1, 1
	}
	if width/height > 16.0/9.0 {
		width = height * 16.0 / 9.0
		x = (size.Width - width) / 2
	} else {
		height = width * 9.0 / 16.0
		y = (size.Height - height) / 2
	}
	return
}

// cameraEnabled selects camera-framing drags instead of title drags.
// It is independent of the active inspector tab, making preview behaviour
// explicit and predictable even when controls are scrolled off-screen.
func (o *TitleOverlay) cameraEnabled() bool {
	return o.studio.inspector != nil && o.studio.inspector.cameraDrag != nil && o.studio.inspector.cameraDrag.Checked
}

func (o *TitleOverlay) layoutGuide() {
	if o.root == nil {
		return
	}
	s := o.studio
	if s.selectedAudio >= 0 || s.selected < 0 || s.selected >= len(s.project.Clips) || s.playing || (!o.cameraEnabled() && s.project.Clips[s.selected].Title == "") {
		o.frame.Hide()
		o.hint.Hide()
		return
	}
	c := s.project.Clips[s.selected]
	if o.cameraEnabled() {
		if !o.moving {
			pose := c.InitialVideoKeyframe(s.titleKeyframeTime(s.selected))
			if s.inspector != nil && s.inspector.cameraZoom != nil {
				pose.Zoom = s.inspector.cameraZoom.Value
				pose.X = s.inspector.cameraX.Value
				pose.Y = s.inspector.cameraY.Value
			}
			o.cameraPose = pose
		}
		vx, vy, vw, vh := o.videoRect()
		zoom := math.Max(1, o.cameraPose.Zoom)
		w, h := vw/float32(zoom), vh/float32(zoom)
		o.frameWidth, o.frameHeight = w, h
		o.frameX = vx + (vw-w)*float32(o.cameraPose.X)
		o.frameY = vy + (vh-h)*float32(o.cameraPose.Y)
		o.frame.Move(fyne.NewPos(o.frameX, o.frameY))
		o.frame.Resize(fyne.NewSize(w, h))
		o.hint.Text = "Drag camera frame"
		if zoom <= 1.001 {
			o.hint.Text = "Zoom above 1× to pan"
		}
		o.hint.Move(fyne.NewPos(o.frameX+5, o.frameY+5))
		o.hint.Resize(o.hint.MinSize())
		o.frame.Show()
		o.hint.Show()
		o.root.Refresh()
		return
	}
	o.hint.Text = "↕ Drag title"
	if !o.moving {
		pose := c.InitialTitleKeyframe(s.titleKeyframeTime(s.selected))
		o.pose = pose
	}
	vx, vy, vw, vh := o.videoRect()
	w := float32(math.Max(85, math.Min(float64(vw*.85), float64(utf8.RuneCountInString(c.Title))*float64(vw)*.042*o.pose.Scale+30)))
	h := float32(math.Max(30, float64(vh)*float64(c.Visual.Size())/720*o.pose.Scale))
	o.frameWidth, o.frameHeight = w, h
	o.frameX = vx + float32(o.pose.X)*vw - w/2
	o.frameY = vy + float32(o.pose.Y)*vh - h/2
	o.frame.Move(fyne.NewPos(o.frameX, o.frameY))
	o.frame.Resize(fyne.NewSize(w, h))
	o.hint.Move(fyne.NewPos(o.frameX+5, o.frameY-15))
	o.hint.Resize(o.hint.MinSize())
	o.frame.Show()
	o.hint.Show()
	o.root.Refresh()
}

func (o *TitleOverlay) MouseDown(e *desktop.MouseEvent) {
	o.canDrag = false
	if e.Button != desktop.MouseButtonPrimary {
		return
	}
	if o.frame == nil || !o.frame.Visible() {
		return
	}
	x, y := e.Position.X, e.Position.Y
	o.cameraDrag = o.cameraEnabled()
	o.canDrag = x >= o.frameX-5 && x <= o.frameX+o.frameWidth+5 && y >= o.frameY-20 && y <= o.frameY+o.frameHeight+5
	if o.cameraDrag && o.cameraPose.Zoom <= 1.001 {
		o.canDrag = false
	}
}
func (o *TitleOverlay) MouseUp(*desktop.MouseEvent) {}

func (o *TitleOverlay) Dragged(e *fyne.DragEvent) {
	if !o.canDrag {
		return
	}
	if !o.moving {
		o.studio.pause()
		o.moving = true
	}
	_, _, vw, vh := o.videoRect()
	if o.cameraDrag {
		zoom := math.Max(1.001, o.cameraPose.Zoom)
		cropWidth := vw - vw/float32(zoom)
		cropHeight := vh - vh/float32(zoom)
		o.cameraPose.X = math.Max(0, math.Min(1, o.cameraPose.X+float64(e.Dragged.DX/cropWidth)))
		o.cameraPose.Y = math.Max(0, math.Min(1, o.cameraPose.Y+float64(e.Dragged.DY/cropHeight)))
		o.layoutGuide()
		return
	}
	o.pose.X = math.Max(.02, math.Min(.98, o.pose.X+float64(e.Dragged.DX/vw)))
	o.pose.Y = math.Max(.02, math.Min(.98, o.pose.Y+float64(e.Dragged.DY/vh)))
	o.layoutGuide()
}
func (o *TitleOverlay) DragEnd() {
	if !o.moving {
		o.canDrag = false
		return
	}
	o.moving = false
	o.canDrag = false
	s := o.studio
	idx := s.selected
	if idx < 0 || idx >= len(s.project.Clips) {
		o.Refresh()
		return
	}
	if o.cameraDrag {
		o.cameraDrag = false
		o.cameraPose.Time = s.titleKeyframeTime(idx)
		s.remember()
		if !s.project.Clips[idx].PutVideoKeyframe(o.cameraPose) {
			s.undo = s.undo[:len(s.undo)-1]
			o.Refresh()
			return
		}
		s.changed("Camera framing keyframe saved at " + timecode(o.cameraPose.Time))
		s.seek(s.playhead)
		return
	}
	o.pose.Time = s.titleKeyframeTime(idx)
	s.remember()
	if !s.project.Clips[idx].PutTitleKeyframe(o.pose) {
		s.undo = s.undo[:len(s.undo)-1]
		o.Refresh()
		return
	}
	s.changed("Moved title · keyframe saved at " + timecode(o.pose.Time))
	s.seek(s.playhead)
}

var _ fyne.Draggable = (*TitleOverlay)(nil)
var _ desktop.Mouseable = (*TitleOverlay)(nil)
