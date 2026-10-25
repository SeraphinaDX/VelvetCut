package main

import (
	"fmt"
	"image/color"
	"math"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

const (
	rulerLeft   float32 = 72
	rulerTop    float32 = 8
	stripTop    float32 = 53
	stripHeight float32 = 84
)

type Timeline struct {
	widget.BaseWidget
	studio  *Studio
	root    *fyne.Container
	mode    string // seek, left, right, move
	index   int
	moveDX  float64
	dragged bool
}

func NewTimeline(studio *Studio) *Timeline {
	t := &Timeline{studio: studio, index: -1}
	t.ExtendBaseWidget(t)
	return t
}

func (t *Timeline) MinSize() fyne.Size {
	width := rulerLeft + float32(t.studio.project.Duration()*t.studio.zoom) + 140
	return fyne.NewSize(float32(math.Max(float64(width), 850)), 165)
}

func (t *Timeline) CreateRenderer() fyne.WidgetRenderer {
	t.root = container.NewWithoutLayout()
	t.redraw()
	return widget.NewSimpleRenderer(t.root)
}
func (t *Timeline) Refresh() {
	if t.root != nil {
		t.redraw()
	}
	t.BaseWidget.Refresh()
}

func addRect(out *[]fyne.CanvasObject, col color.Color, x, y, w, h float32) {
	if w <= 0 || h <= 0 {
		return
	}
	r := canvas.NewRectangle(col)
	r.Move(fyne.NewPos(x, y))
	r.Resize(fyne.NewSize(w, h))
	*out = append(*out, r)
}
func addLabel(out *[]fyne.CanvasObject, label string, col color.Color, x, y, size float32) {
	txt := canvas.NewText(label, col)
	txt.TextSize = size
	txt.Move(fyne.NewPos(x, y))
	*out = append(*out, txt)
}
func addLine(out *[]fyne.CanvasObject, col color.Color, x1, y1, x2, y2, width float32) {
	line := canvas.NewLine(col)
	line.Position1 = fyne.NewPos(x1, y1)
	line.Position2 = fyne.NewPos(x2, y2)
	line.StrokeWidth = width
	*out = append(*out, line)
}
func (t *Timeline) redraw() {
	if t.root == nil {
		return
	}
	s := t.studio
	width := t.MinSize().Width
	deep := color.NRGBA{R: 24, G: 24, B: 38, A: 255}
	track := color.NRGBA{R: 32, G: 34, B: 48, A: 255}
	subdued := color.NRGBA{R: 132, G: 138, B: 156, A: 255}
	border := color.NRGBA{R: 73, G: 77, B: 103, A: 255}
	objects := make([]fyne.CanvasObject, 0, 20+len(s.project.Clips)*9)
	addRect(&objects, deep, 0, 0, width, 165)
	addRect(&objects, track, 0, stripTop-4, width, stripHeight+8)
	addRect(&objects, color.NRGBA{R: 35, G: 38, B: 54, A: 255}, 0, 0, rulerLeft, 165)
	addLabel(&objects, "VIDEO 1", pale, 9, stripTop+22, 11)
	addLabel(&objects, "SEC", subdued, 12, 14, 11)
	addLine(&objects, border, 0, stripTop-4, width, stripTop-4, 1)
	addLine(&objects, border, 0, stripTop+stripHeight+4, width, stripTop+stripHeight+4, 1)
	total := s.project.Duration()
	tickStep := math.Max(1, math.Ceil(total/250))
	for sec := 0.0; sec <= total+0.001; sec += tickStep {
		x := rulerLeft + float32(sec*s.zoom)
		major := math.Mod(sec, 5*tickStep) < 0.0001
		size := float32(7)
		if major {
			size = 15
			addLabel(&objects, timecode(sec), pale, x+3, rulerTop, 10)
		}
		addLine(&objects, subdued, x, stripTop-8-size, x, stripTop-8, 1)
	}
	var elapsed float64
	for index, clip := range s.project.Clips {
		x := rulerLeft + float32(elapsed*s.zoom)
		w := float32(clip.Duration() * s.zoom)
		col := color.NRGBA{R: 65, G: 79, B: 128, A: 255}
		if index%2 == 1 {
			col = color.NRGBA{R: 65, G: 98, B: 134, A: 255}
		}
		if index == s.selected {
			col = color.NRGBA{R: 119, G: 80, B: 162, A: 255}
		}
		addRect(&objects, col, x, stripTop, w, stripHeight)
		if thumb, ok := s.thumbs[clip.Path]; ok && w > 32 {
			photo := canvas.NewImageFromImage(thumb)
			photo.FillMode = canvas.ImageFillCover
			photo.Move(fyne.NewPos(x+3, stripTop+25))
			photo.Resize(fyne.NewSize(float32(math.Max(0, float64(w-6))), stripHeight-28))
			objects = append(objects, photo)
		}
		// Sparse amplitude envelope over the lower half of each video clip.
		// Source-range offsets keep peaks aligned during left/right trimming.
		if peaks, ok := s.waveforms[clip.Path]; ok && len(peaks) > 0 && w > 14 && clip.SourceDuration > 0 {
			step := float32(4)
			mid := stripTop + stripHeight - 19
			maxh := float32(15)
			for px := float32(5); px < w-5; px += step {
				source := clip.SourceStart + float64(px/w)*clip.Duration()
				idx := int(source / clip.SourceDuration * float64(len(peaks)))
				if idx < 0 {
					idx = 0
				}
				if idx >= len(peaks) {
					idx = len(peaks) - 1
				}
				amp := float32(math.Sqrt(float64(peaks[idx]))) * maxh
				addLine(&objects, color.NRGBA{R: 181, G: 230, B: 234, A: 240}, x+px, mid-amp, x+px, mid+amp, 2)
			}
		}
		addRect(&objects, color.NRGBA{R: 31, G: 27, B: 56, A: 220}, x+2, stripTop+2, w-4, 22)
		if w > 46 {
			name := filepath.Base(clip.Name)
			if len(name) > 30 {
				name = name[:27] + "…"
			}
			if w < 115 && len(name) > 10 {
				name = name[:9] + "…"
			}
			addLabel(&objects, name, color.White, x+8, stripTop+5, 12)
		}
		if clip.Title != "" && w > 60 {
			addLabel(&objects, "T · title", color.NRGBA{R: 255, G: 234, B: 170, A: 255}, x+8, stripTop+25, 11)
		}
		if w > 100 {
			addLabel(&objects, timecode(clip.Duration()), color.White, x+8, stripTop+stripHeight-19, 11)
		}
		// Grips: editing the left grip advances/rewinds source in; right edits out.
		grip := float32(math.Min(float64(w/3), 8))
		addRect(&objects, color.NRGBA{R: 225, G: 204, B: 254, A: 255}, x, stripTop, grip, stripHeight)
		addRect(&objects, color.NRGBA{R: 225, G: 204, B: 254, A: 255}, x+w-grip, stripTop, grip, stripHeight)
		if s.selected == index {
			addLine(&objects, purple, x, stripTop, x+w, stripTop, 2)
			addLine(&objects, purple, x, stripTop+stripHeight, x+w, stripTop+stripHeight, 2)
		}
		elapsed += clip.Duration()
	}
	// Visible cut boundaries make snap targets clear while editing.
	elapsed = 0
	for _, clip := range s.project.Clips {
		elapsed += clip.Duration()
		x := rulerLeft + float32(elapsed*s.zoom)
		addLine(&objects, color.NRGBA{R: 223, G: 201, B: 249, A: 180}, x, stripTop-6, x, stripTop+stripHeight+6, 1)
	}
	if len(s.project.Clips) == 0 {
		addLabel(&objects, "Your movie starts here · Import, then insert clips", subdued, rulerLeft+22, stripTop+24, 14)
	}
	if t.mode == "move" && t.dragged && t.index >= 0 && t.index < len(s.project.Clips) {
		start := s.project.Start(t.index) + s.project.Clips[t.index].Duration()/2 + t.moveDX/s.zoom
		x := rulerLeft + float32(start*s.zoom)
		addLine(&objects, purple, x, stripTop-7, x, stripTop+stripHeight+7, 3)
	}
	cursorX := rulerLeft + float32(s.playhead*s.zoom)
	cursorColor := color.NRGBA{R: 255, G: 101, B: 121, A: 255}
	addLine(&objects, cursorColor, cursorX, rulerTop, cursorX, stripTop+stripHeight+14, 2)
	addRect(&objects, cursorColor, cursorX-5, stripTop-12, 10, 9)
	t.root.Objects = objects
	t.root.Refresh()
}

func (t *Timeline) clipAt(x float32) (index int, localX float32) {
	elapsed := float32(rulerLeft)
	for i, c := range t.studio.project.Clips {
		end := elapsed + float32(c.Duration()*t.studio.zoom)
		if x >= elapsed && x < end {
			return i, x - elapsed
		}
		elapsed = end
	}
	return -1, 0
}

func (t *Timeline) MouseDown(e *desktop.MouseEvent) {
	if e.Button != desktop.MouseButtonPrimary {
		return
	}
	s := t.studio
	s.pause()
	t.index = -1
	t.mode = ""
	t.moveDX = 0
	t.dragged = false
	if e.Position.Y < stripTop || e.Position.Y > stripTop+stripHeight {
		t.mode = "seek"
		s.seek(s.snapTime((float64(e.Position.X - rulerLeft)) / s.zoom))
		return
	}
	idx, local := t.clipAt(e.Position.X)
	if idx < 0 {
		return
	}
	t.index = idx
	s.selected = idx
	w := float32(s.project.Clips[idx].Duration() * s.zoom)
	if local <= 10 {
		t.mode = "left"
		s.remember()
	} else if w-local <= 10 {
		t.mode = "right"
		s.remember()
	} else {
		t.mode = "move"
		s.seek(s.snapTime((float64(e.Position.X - rulerLeft)) / s.zoom))
	}
	s.changed("Selected " + s.project.Clips[idx].Name)
}

func (t *Timeline) MouseUp(e *desktop.MouseEvent) {
	if t.mode == "seek" {
		t.studio.seek(t.studio.snapTime(float64(e.Position.X-rulerLeft) / t.studio.zoom))
	}
	// For a drag the actual edit/reorder is committed in DragEnd.
	if !t.dragged {
		t.mode = ""
	}
}

func (t *Timeline) Dragged(e *fyne.DragEvent) {
	s := t.studio
	delta := float64(e.Dragged.DX) / s.zoom
	if delta == 0 {
		return
	}
	t.dragged = true
	switch t.mode {
	case "seek":
		s.seek(s.snapTime(float64(e.Position.X-rulerLeft) / s.zoom))
	case "left":
		if err := s.project.TrimLeft(t.index, delta); err == nil {
			s.changed("Trimming in-point")
		}
	case "right":
		if err := s.project.TrimRight(t.index, delta); err == nil {
			s.changed("Trimming out-point")
		}
	case "move":
		t.moveDX += float64(e.Dragged.DX)
		s.info.SetText(fmt.Sprintf("Repositioning %s…", s.project.Clips[t.index].Name))
		t.Refresh()
	}
}

func (t *Timeline) DragEnd() {
	s := t.studio
	if t.mode == "move" && t.dragged && t.index >= 0 && t.index < len(s.project.Clips) {
		from := t.index
		center := s.project.Start(from) + s.project.Clips[from].Duration()/2 + t.moveDX/s.zoom
		destination := 0
		var cursor float64
		for i, c := range s.project.Clips {
			if i == from {
				continue
			}
			if center > cursor+c.Duration()/2 {
				destination++
			}
			cursor += c.Duration()
		}
		if destination != from {
			s.remember()
			_ = s.project.Move(from, destination)
			s.selected = destination
			s.changed("Reordered clips")
		}
	}
	t.mode = ""
	t.dragged = false
	t.moveDX = 0
	t.Refresh()
}

var _ desktop.Mouseable = (*Timeline)(nil)
var _ fyne.Draggable = (*Timeline)(nil)
