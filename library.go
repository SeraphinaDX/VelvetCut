package main

import (
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// LibraryRow supplies an in-app drag source. It stays anchored in widget.List;
// the drag position is used only to choose where to insert the selected clip.
type LibraryRow struct {
	widget.BaseWidget
	studio        *Studio
	index         int
	name, details *widget.Label
	dragged       bool
	lastPos       fyne.Position
}

func newLibraryRow(s *Studio) *LibraryRow {
	row := &LibraryRow{studio: s, index: -1, name: widget.NewLabel("Name"), details: widget.NewLabel("Duration")}
	row.ExtendBaseWidget(row)
	return row
}
func (r *LibraryRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewVBox(r.name, r.details))
}
func (r *LibraryRow) setClip(index int) {
	r.index = index
	if index < 0 || index >= len(r.studio.library) {
		return
	}
	clip := r.studio.library[index]
	r.name.SetText("☷  " + clip.Name)
	r.details.SetText(fmt.Sprintf("%s · %s · drag to timeline", timecode(clip.Duration()), filepath.Ext(clip.Path)))
}
func (r *LibraryRow) MouseDown(e *desktop.MouseEvent) {
	if e.Button == desktop.MouseButtonPrimary && r.index >= 0 && r.index < len(r.studio.library) {
		r.studio.mediaList.Select(r.index)
	}
}
func (r *LibraryRow) MouseUp(*desktop.MouseEvent) {}
func (r *LibraryRow) Dragged(e *fyne.DragEvent) {
	r.dragged = true
	r.lastPos = e.AbsolutePosition
}
func (r *LibraryRow) DragEnd() {
	if r.dragged {
		r.studio.dropLibraryClip(r.index, r.lastPos)
	}
	r.dragged = false
}

var _ desktop.Mouseable = (*LibraryRow)(nil)
var _ fyne.Draggable = (*LibraryRow)(nil)
