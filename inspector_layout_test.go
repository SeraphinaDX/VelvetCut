package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/SeraphinaDX/VelvetCut/editor"
)

// A clip name or inspector field must not increase the minimum workspace width
// and shove the media library / preview outside the window on selection.
func TestSelectingLongNamedClipDoesNotGrowWorkspace(t *testing.T) {
	app := test.NewTempApp(t)
	win := app.NewWindow("VelvetCut layout")
	s := &Studio{
		window: win, project: editor.NewProject(), selected: -1,
		selectedAudio: -1, libraryChoice: -1, zoom: 45,
	}
	content := s.build()
	win.SetContent(content)
	before := content.MinSize().Width

	name := strings.Repeat("my-extraordinarily-long-video-filename-", 20) + ".mp4"
	s.project.Clips = []editor.Clip{{
		ID: "clip-1", Name: name, Path: "/tmp/" + name,
		SourceStart: 0, SourceEnd: 12, SourceDuration: 12,
	}}
	s.selected = 0
	s.inspector.loadSelection()
	s.selectedInfo.SetText(name + " · Source 00:00 – 00:12")
	s.info.SetText("Selected " + name)

	after := content.MinSize().Width
	if after > before+1 {
		t.Fatalf("selecting a long-named clip expanded the workspace minimum from %.1f to %.1f", before, after)
	}
}
