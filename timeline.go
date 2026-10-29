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
	rulerLeft       float32 = 72
	rulerTop        float32 = 8
	stripTop        float32 = 53
	stripHeight     float32 = 84
	audioTop        float32 = 159
	audioLaneHeight float32 = 43
	audioLaneGap    float32 = 4
	timelineHeight  float32 = 318
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
	for _, a := range t.studio.project.AudioClips {
		width = float32(math.Max(float64(width), float64(rulerLeft)+float64((a.Start+a.Duration())*t.studio.zoom)+140))
	}
	return fyne.NewSize(float32(math.Max(float64(width), 850)), timelineHeight)
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
	addRect(&objects, deep, 0, 0, width, timelineHeight)
	addRect(&objects, track, 0, stripTop-4, width, stripHeight+8)
	addRect(&objects, color.NRGBA{R: 35, G: 38, B: 54, A: 255}, 0, 0, rulerLeft, timelineHeight)
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
	starts := make([]float64, len(s.project.Clips))
	var clipStart float64
	for index, clip := range s.project.Clips {
		starts[index] = clipStart
		x := rulerLeft + float32(clipStart*s.zoom)
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
			step := float32(math.Max(4, float64(w/200)))
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
		if clip.Visual.Active() && w > 70 {
			addLabel(&objects, "FX", color.NRGBA{R: 245, G: 198, B: 124, A: 255}, x+w-26, stripTop+25, 11)
		}
		if clip.Title != "" && w > 60 {
			addLabel(&objects, "T · title", color.NRGBA{R: 255, G: 234, B: 170, A: 255}, x+8, stripTop+25, 11)
		}
		// Gold notches show each title keyframe at its clip-local time.
		for _, k := range clip.TitleKeyframes {
			at := x + float32(k.Time*s.zoom)
			addRect(&objects, color.NRGBA{R: 255, G: 216, B: 100, A: 255}, at-2, stripTop+22, 4, 9)
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
		clipStart += clip.Duration() - clip.TransitionOut.Duration
	}
	// A patterned translucent transition region marks actual overlap time.
	for i, clip := range s.project.Clips {
		if i+1 == len(s.project.Clips) {
			break
		}
		d := clip.TransitionOut.Duration
		if d <= 0 {
			cut := starts[i+1] + d/2
			x := rulerLeft + float32(cut*s.zoom)
			addLine(&objects, color.NRGBA{R: 223, G: 201, B: 249, A: 180}, x, stripTop-6, x, stripTop+stripHeight+6, 1)
			continue
		}
		x := rulerLeft + float32(starts[i+1]*s.zoom)
		w := float32(d * s.zoom)
		addRect(&objects, color.NRGBA{R: 49, G: 25, B: 86, A: 235}, x, stripTop, w, stripHeight)
		for stripe := float32(4); stripe < w-2; stripe += 10 {
			addLine(&objects, color.NRGBA{R: 201, G: 160, B: 251, A: 185}, x+stripe, stripTop+4, x+stripe, stripTop+stripHeight-4, 1)
		}
		tag := "⇄"
		if clip.TransitionOut.Kind == "black" {
			tag = "◈"
		}
		addLabel(&objects, tag, pale, x+w/2-6, stripTop+20, 16)
		if w >= 55 {
			addLabel(&objects, fmt.Sprintf("%.2fs", d), pale, x+6, stripTop+52, 11)
		}
		addLine(&objects, purple, x, stripTop-5, x, stripTop+stripHeight+5, 2)
		addLine(&objects, purple, x+w, stripTop-5, x+w, stripTop+stripHeight+5, 2)
	}

	// Independent audio lanes: waveforms and audio grips occupy tracks below
	// video. Overlapping sounds on different lanes remain separately editable.
	for lane := 0; lane < 3; lane++ {
		y := audioTop + float32(lane)*(audioLaneHeight+audioLaneGap)
		addRect(&objects, color.NRGBA{R: 26, G: 46, B: 55, A: 255}, 0, y, width, audioLaneHeight)
		addLabel(&objects, fmt.Sprintf("AUDIO %d", lane+1), pale, 8, y+13, 11)
		for idx, a := range s.project.AudioClips {
			if a.Track != lane {
				continue
			}
			x := rulerLeft + float32(a.Start*s.zoom)
			w := float32(a.Duration() * s.zoom)
			c := color.NRGBA{R: 31, G: 112, B: 106, A: 255}
			if idx == s.selectedAudio {
				c = color.NRGBA{R: 92, G: 151, B: 123, A: 255}
			}
			addRect(&objects, c, x, y+2, w, audioLaneHeight-4)
			if peaks, ok := s.waveforms[a.Path]; ok && len(peaks) > 0 && a.SourceDuration > 0 && w > 14 {
				step := float32(math.Max(3, float64(w/200)))
				for px := float32(4); px < w-3; px += step {
					source := a.SourceStart + float64(px/w)*a.Duration()
					index := int(source / a.SourceDuration * float64(len(peaks)))
					if index < 0 {
						index = 0
					}
					if index >= len(peaks) {
						index = len(peaks) - 1
					}
					h := float32(math.Sqrt(float64(peaks[index]))) * 12
					addLine(&objects, color.NRGBA{R: 182, G: 248, B: 225, A: 255}, x+px, y+audioLaneHeight-15-h, x+px, y+audioLaneHeight-15+h, 1)
				}
			}
			if w > 30 {
				name := a.Name
				if len(name) > 24 {
					name = name[:21] + "…"
				}
				addLabel(&objects, name, color.White, x+8, y+3, 10)
			}
			grip := float32(math.Min(7, float64(w/3)))
			addRect(&objects, color.NRGBA{R: 162, G: 238, B: 226, A: 255}, x, y+2, grip, audioLaneHeight-4)
			addRect(&objects, color.NRGBA{R: 162, G: 238, B: 226, A: 255}, x+w-grip, y+2, grip, audioLaneHeight-4)
		}
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
	addLine(&objects, cursorColor, cursorX, rulerTop, cursorX, timelineHeight-8, 2)
	addRect(&objects, cursorColor, cursorX-5, stripTop-12, 10, 9)
	t.root.Objects = objects
	t.root.Refresh()
}

func (t *Timeline) audioAt(x, y float32) (int, float32) {
	for i := len(t.studio.project.AudioClips) - 1; i >= 0; i-- {
		a := t.studio.project.AudioClips[i]
		row := audioTop + float32(a.Track)*(audioLaneHeight+audioLaneGap)
		start := rulerLeft + float32(a.Start*t.studio.zoom)
		end := start + float32(a.Duration()*t.studio.zoom)
		if y >= row && y < row+audioLaneHeight && x >= start && x < end {
			return i, x - start
		}
	}
	return -1, 0
}

func (t *Timeline) transitionAt(x float32) int {
	s := t.studio
	for i, c := range s.project.Clips {
		if i+1 >= len(s.project.Clips) || !c.TransitionOut.Active() {
			continue
		}
		start := rulerLeft + float32(s.project.Start(i+1)*s.zoom)
		w := float32(c.TransitionOut.Duration * s.zoom)
		if x >= start-3 && x <= start+w+3 {
			return i
		}
	}
	return -1
}
func (t *Timeline) clipAt(x float32) (index int, localX float32) {
	// Last drawn incoming clip is on top of earlier overlapping clips.
	for i := len(t.studio.project.Clips) - 1; i >= 0; i-- {
		c := t.studio.project.Clips[i]
		start := rulerLeft + float32(t.studio.project.Start(i)*t.studio.zoom)
		end := start + float32(c.Duration()*t.studio.zoom)
		if x >= start && x < end {
			return i, x - start
		}
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
	if e.Position.Y >= audioTop {
		idx, local := t.audioAt(e.Position.X, e.Position.Y)
		if idx < 0 {
			return
		}
		t.index = idx
		s.selected = -1
		s.selectedAudio = idx
		w := float32(s.project.AudioClips[idx].Duration() * s.zoom)
		if local <= 10 {
			t.mode = "audio-left"
		} else if w-local <= 10 {
			t.mode = "audio-right"
		} else {
			t.mode = "audio-move"
		}
		s.remember()
		s.changed("Selected audio: " + s.project.AudioClips[idx].Name)
		return
	}
	if e.Position.Y < stripTop || e.Position.Y > stripTop+stripHeight {
		t.mode = "seek"
		s.seek(s.snapTime((float64(e.Position.X - rulerLeft)) / s.zoom))
		return
	}
	if idx := t.transitionAt(e.Position.X); idx >= 0 {
		t.index = idx
		s.selected = idx
		s.selectedAudio = -1
		t.mode = "transition"
		s.remember()
		s.changed("Drag left/right to adjust transition length")
		return
	}
	idx, local := t.clipAt(e.Position.X)
	if idx < 0 {
		return
	}
	t.index = idx
	s.selected = idx
	s.selectedAudio = -1
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
	case "audio-left":
		if err := s.project.TrimAudioLeft(t.index, delta); err == nil {
			s.changed("Trimming audio start")
		}
	case "audio-right":
		if err := s.project.TrimAudioRight(t.index, delta); err == nil {
			s.changed("Trimming audio end")
		}
	case "audio-move":
		if t.index >= 0 && t.index < len(s.project.AudioClips) {
			a := s.project.AudioClips[t.index]
			if err := s.project.MoveAudio(t.index, a.Start+delta); err == nil {
				s.changed("Moving audio along timeline")
			}
		}
	case "seek":
		s.seek(s.snapTime(float64(e.Position.X-rulerLeft) / s.zoom))
	case "transition":
		if t.index >= 0 && t.index+1 < len(s.project.Clips) {
			trans := s.project.Clips[t.index].TransitionOut
			next := trans.Duration + delta
			if next < 0.1 {
				_ = s.project.SetTransition(t.index, "", 0)
			} else if err := s.project.SetTransition(t.index, trans.Kind, next); err == nil {
				s.changed("Adjusting " + trans.Kind + " transition")
			}
		}
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
		for i, c := range s.project.Clips {
			if i == from {
				continue
			}
			if center > s.project.Start(i)+c.Duration()/2 {
				destination++
			}
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
