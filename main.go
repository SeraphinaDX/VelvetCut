package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/SeraphinaDX/VelvetCut/editor"
	"github.com/SeraphinaDX/VelvetCut/media"
)

var (
	purple = color.NRGBA{R: 183, G: 138, B: 248, A: 255}
	pale   = color.NRGBA{R: 221, G: 218, B: 237, A: 255}
)

type violetTheme struct{ fyne.Theme }

func (t violetTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNamePrimary {
		return purple
	}
	return t.Theme.Color(name, variant)
}

type Studio struct {
	window                  fyne.Window
	project                 editor.Project
	library                 []editor.Clip
	libraryChoice           int
	selected                int
	playhead                float64
	zoom                    float64 // logical pixels per second
	timeline                *Timeline
	timelineScroll          *container.Scroll
	mediaList               *widget.List
	info                    *widget.Label
	clock                   *widget.Label
	selectedInfo            *widget.Label
	preview                 *canvas.Image
	playButton              *widget.Button
	projectPath             string
	undo                    []editor.Project
	redo                    []editor.Project
	playing                 bool
	playbackStop            chan struct{}
	previewCancel           context.CancelFunc
	previewSeq              uint64
	lastFrameKey            string
	frames                  map[string]image.Image
	thumbs                  map[string]image.Image
	waveforms               map[string][]float32
	player                  *media.Player
	playerGeneration        uint64
	nextID                  uint64
	snapping                bool
	embeddedPreview         bool
	previewVideoClip        int
	previewVideoStart       float64
	previewVideoSourceStart float64
	previewVideoFrames      <-chan media.PreviewFrame
	previewVideoNext        *media.PreviewFrame
	previewVideoCancel      context.CancelFunc
	previewNotes            *widget.Label
}

// Canvas shortcuts run whenever a text entry is not consuming keyboard input.
// Ctrl combinations are explicit desktop shortcuts; typing in a title dialog
// therefore never accidentally splits or deletes a clip.
func (s *Studio) installShortcuts() {
	canvas := s.window.Canvas()
	canvas.SetOnTypedRune(func(r rune) {
		switch r {
		case ' ':
			s.togglePlayback()
		case 's', 'S':
			s.split()
		}
	})
	canvas.SetOnTypedKey(func(e *fyne.KeyEvent) {
		switch e.Name {
		case fyne.KeyDelete, fyne.KeyBackspace:
			s.deleteSelected()
		case fyne.KeyLeft:
			s.pause()
			s.seek(s.snapTime(s.playhead - 1.0/30))
		case fyne.KeyRight:
			s.pause()
			s.seek(s.snapTime(s.playhead + 1.0/30))
		case fyne.KeyHome:
			s.pause()
			s.seek(0)
		case fyne.KeyEnd:
			s.pause()
			s.seek(s.project.Duration())
		}
	})
	for _, binding := range []struct {
		key    fyne.KeyName
		action func()
	}{
		{fyne.KeyZ, s.undoEdit}, {fyne.KeyY, s.redoEdit}, {fyne.KeyS, s.saveProject},
		{fyne.KeyO, s.openProject},
	} {
		shortcut := &desktop.CustomShortcut{KeyName: binding.key, Modifier: fyne.KeyModifierControl}
		action := binding.action
		canvas.AddShortcut(shortcut, func(fyne.Shortcut) { action() })
	}
}

func main() {
	a := app.NewWithID("ca.cerberusgames.velvetcut")
	a.Settings().SetTheme(violetTheme{Theme: theme.DarkTheme()})
	win := a.NewWindow("VelvetCut · Video Editor")
	s := &Studio{
		window: win, project: editor.NewProject(), selected: -1, libraryChoice: -1,
		zoom: 45, snapping: true, embeddedPreview: true, previewVideoClip: -1,
		frames: make(map[string]image.Image), thumbs: make(map[string]image.Image), waveforms: make(map[string][]float32),
	}
	win.Resize(fyne.NewSize(1240, 800))
	win.SetContent(s.build())
	s.installShortcuts()
	win.SetOnDropped(s.importDroppedFiles)
	win.SetOnClosed(func() {
		s.stopPlayer()
		s.stopVideoStream()
		if s.previewCancel != nil {
			s.previewCancel()
		}
	})
	if err := media.CheckTools(); err != nil {
		dialog.ShowInformation("FFmpeg needed", "Install ffmpeg and ffprobe, then restart VelvetCut.\n\n"+err.Error(), win)
	}
	win.ShowAndRun()
}

func (s *Studio) build() fyne.CanvasObject {
	s.info = widget.NewLabel("Ready · Import a video to begin")
	s.clock = widget.NewLabel("00:00.00 / 00:00.00")
	s.selectedInfo = widget.NewLabel("Select a clip to inspect its source range")
	s.preview = canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 960, 540)))
	s.preview.SetMinSize(fyne.NewSize(350, 215))
	s.preview.FillMode = canvas.ImageFillContain
	s.timeline = NewTimeline(s)
	s.timelineScroll = container.NewHScroll(s.timeline)
	s.timelineScroll.SetMinSize(fyne.NewSize(400, 165))

	s.mediaList = widget.NewList(
		func() int { return len(s.library) },
		func() fyne.CanvasObject { return newLibraryRow(s) },
		func(id widget.ListItemID, item fyne.CanvasObject) { item.(*LibraryRow).setClip(id) },
	)
	s.mediaList.OnSelected = func(id widget.ListItemID) { s.libraryChoice = id }
	mediaPanel := container.NewBorder(
		container.NewVBox(widget.NewLabelWithStyle("MEDIA LIBRARY", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Drop files here · drag clips to timeline"), widget.NewButtonWithIcon("Import video", theme.FolderOpenIcon(), s.importFile)),
		container.NewVBox(widget.NewButtonWithIcon("Insert at playhead", theme.ContentAddIcon(), s.insertClip),
			widget.NewLabel("Insert uses the red playhead position.")), nil, nil, s.mediaList)

	s.playButton = widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), s.togglePlayback)
	s.previewNotes = widget.NewLabel("In-editor playback: 15 fps preview; mpv synchronizes audio")
	previewMode := widget.NewCheck("Play video in editor (beta)", func(on bool) {
		s.stopPlayer()
		s.embeddedPreview = on
		if on {
			s.previewNotes.SetText("In-editor playback: 15 fps preview; mpv synchronizes audio")
		} else {
			s.previewNotes.SetText("Classic playback: mpv opens a separate video window")
		}
	})
	previewMode.SetChecked(true)
	previewPanel := container.NewBorder(
		container.NewVBox(widget.NewLabelWithStyle("PREVIEW", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), previewMode, s.previewNotes),
		container.NewVBox(container.NewCenter(container.NewHBox(
			widget.NewButtonWithIcon("Start", theme.MediaSkipPreviousIcon(), func() { s.pause(); s.seek(0) }),
			s.playButton,
			widget.NewButtonWithIcon("End", theme.MediaSkipNextIcon(), func() { s.pause(); s.seek(s.project.Duration()) }),
			s.clock,
		)), s.selectedInfo), nil, nil,
		container.NewMax(canvas.NewRectangle(color.NRGBA{R: 11, G: 12, B: 21, A: 255}), s.preview))

	editingToolbar := container.NewHBox(
		widget.NewButtonWithIcon("Split at playhead", theme.ContentCutIcon(), s.split),
		widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), s.deleteSelected),
		widget.NewButton("Title & fades", s.editAppearance),
		widget.NewButtonWithIcon("Undo", theme.ContentUndoIcon(), s.undoEdit),
		widget.NewButtonWithIcon("Redo", theme.ContentRedoIcon(), s.redoEdit),
		widget.NewSeparator(), widget.NewLabel("Zoom"),
	)
	slider := widget.NewSlider(20, 140)
	slider.Value = s.zoom
	slider.Step = 5
	slider.OnChanged = func(v float64) { s.zoom = v; s.timeline.Refresh(); s.timelineScroll.Refresh() }
	snapCheck := widget.NewCheck("Snap to cuts", func(on bool) { s.snapping = on })
	snapCheck.SetChecked(true)
	zoomLine := container.NewBorder(nil, nil, container.NewHBox(editingToolbar, snapCheck), nil, slider)
	timelinePanel := container.NewBorder(
		container.NewVBox(widget.NewLabelWithStyle("TIMELINE · VIDEO 1", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), zoomLine),
		widget.NewLabel("Drop library clips to insert  •  S split  •  Space play/pause  •  Drag edges to trim  •  Drag body to reorder"),
		nil, nil, s.timelineScroll)
	topPanel := container.NewHSplit(mediaPanel, previewPanel)
	topPanel.Offset = 0.27
	workspace := container.NewVSplit(topPanel, timelinePanel)
	workspace.Offset = 0.64
	header := container.NewBorder(nil, nil,
		container.NewHBox(widget.NewLabelWithStyle("✦ VELVETCUT", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), widget.NewSeparator()),
		container.NewHBox(
			widget.NewButtonWithIcon("New", theme.FileIcon(), s.newProject),
			widget.NewButtonWithIcon("Open", theme.FolderOpenIcon(), s.openProject),
			widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), s.saveProject),
			widget.NewButtonWithIcon("Export MP4", theme.DownloadIcon(), s.exportMovie),
		), widget.NewLabel("Non-destructive movie editor"))
	return container.NewBorder(header, container.NewHBox(widget.NewLabel("VelvetCut 0.3.0"), s.info), nil, nil, workspace)
}

func (s *Studio) id() string {
	s.nextID++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), s.nextID)
}

func timecode(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	minutes := int(sec) / 60
	return fmt.Sprintf("%02d:%05.2f", minutes, sec-float64(minutes*60))
}

func (s *Studio) importFile() {
	dlg := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			s.problem(err)
			return
		}
		if r == nil {
			return
		}
		path := r.URI().Path()
		_ = r.Close()
		s.importPath(path, nil)
	}, s.window)
	dlg.SetFilter(storage.NewExtensionFileFilter([]string{".mp4", ".mov", ".mkv", ".webm", ".avi", ".m4v"}))
	dlg.Show()
}

// importPath registers a video in the library. When fromTimeline is specified
// it is also ripple-inserted at the drop time after metadata probing finishes.
func (s *Studio) importPath(path string, fromTimeline *float64) {
	if !isVideoPath(path) {
		s.info.SetText("Unsupported video file: " + filepath.Base(path))
		return
	}
	s.info.SetText("Inspecting " + filepath.Base(path) + "…")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		meta, err := media.Probe(ctx, path)
		fyne.Do(func() {
			if err != nil {
				s.problem(err)
				return
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				s.problem(err)
				return
			}
			c := editor.Clip{ID: s.id(), Path: abs, Name: filepath.Base(abs), SourceStart: 0, SourceEnd: meta.Duration, SourceDuration: meta.Duration, HasAudio: meta.HasAudio}
			s.library = append(s.library, c)
			s.libraryChoice = len(s.library) - 1
			s.mediaList.Refresh()
			s.mediaList.Select(s.libraryChoice)
			s.makeThumbnail(c)
			s.makeWaveform(c)
			if fromTimeline != nil {
				s.insertAt(*fromTimeline, c)
			} else {
				s.info.SetText("Imported " + c.Name + " · drag to timeline or choose Insert")
			}
		})
	}()
}

func isVideoPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".mov", ".mkv", ".webm", ".avi", ".m4v":
		return true
	}
	return false
}

func (s *Studio) importDroppedFiles(pos fyne.Position, uris []fyne.URI) {
	at, overTimeline := s.timelineDropTime(pos)
	for _, uri := range uris {
		if uri.Scheme() != "file" {
			continue
		}
		path := uri.Path()
		if !isVideoPath(path) {
			continue
		}
		if overTimeline {
			drop := at
			s.importPath(path, &drop)
		} else {
			s.importPath(path, nil)
		}
	}
}

// A timeline's absolute position accounts for horizontal scrolling; the scroll
// viewport is used to reject drops outside its visible rectangle.
func (s *Studio) timelineDropTime(pos fyne.Position) (float64, bool) {
	viewport := fyne.CurrentApp().Driver().AbsolutePositionForObject(s.timelineScroll)
	size := s.timelineScroll.Size()
	if pos.X < viewport.X || pos.X > viewport.X+size.Width || pos.Y < viewport.Y || pos.Y > viewport.Y+size.Height {
		return 0, false
	}
	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(s.timeline)
	at := float64(pos.X-origin.X-rulerLeft) / s.zoom
	at = math.Max(0, math.Min(at, s.project.Duration()))
	return s.snapTime(at), true
}

func (s *Studio) dropLibraryClip(index int, pos fyne.Position) {
	if index < 0 || index >= len(s.library) {
		return
	}
	at, ok := s.timelineDropTime(pos)
	if !ok {
		s.info.SetText("Drop a library clip onto the video timeline to insert it")
		return
	}
	c := s.library[index]
	s.insertAt(at, c)
}

func (s *Studio) snapTime(at float64) float64 {
	if !s.snapping || s.zoom <= 0 {
		return at
	}
	return s.project.SnapTime(at, 8/s.zoom)
}

func (s *Studio) makeThumbnail(c editor.Clip) {
	if _, ok := s.thumbs[c.Path]; ok {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		img, err := media.Frame(ctx, c.Path, math.Min(c.SourceDuration/2, 5))
		if err == nil {
			fyne.Do(func() { s.thumbs[c.Path] = img; s.timeline.Refresh() })
		}
	}()
}

func (s *Studio) makeWaveform(c editor.Clip) {
	if !c.HasAudio {
		return
	}
	if _, ok := s.waveforms[c.Path]; ok {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		peaks, err := media.Waveform(ctx, c.Path, c.SourceDuration)
		if err == nil {
			fyne.Do(func() { s.waveforms[c.Path] = peaks; s.timeline.Refresh() })
		}
	}()
}

func fadeValue(value string) float64 {
	switch value {
	case "0.5 seconds":
		return 0.5
	case "1 second":
		return 1
	default:
		return 0
	}
}
func fadeName(value float64) string {
	if value >= 0.95 {
		return "1 second"
	}
	if value >= 0.45 {
		return "0.5 seconds"
	}
	return "None"
}
func (s *Studio) editAppearance() {
	if s.selected < 0 || s.selected >= len(s.project.Clips) {
		dialog.ShowInformation("Select a clip", "Click a clip on the timeline first.", s.window)
		return
	}
	clip := s.project.Clips[s.selected]
	title := widget.NewEntry()
	title.SetText(clip.Title)
	title.SetPlaceHolder("Optional on-screen title (first 3 seconds)")
	fadeIn := widget.NewSelect([]string{"None", "0.5 seconds", "1 second"}, nil)
	fadeIn.SetSelected(fadeName(clip.FadeIn))
	fadeOut := widget.NewSelect([]string{"None", "0.5 seconds", "1 second"}, nil)
	fadeOut.SetSelected(fadeName(clip.FadeOut))
	dialog.NewForm("Clip title and fade from/to black", "Apply", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Title", title), widget.NewFormItem("Fade in", fadeIn), widget.NewFormItem("Fade out", fadeOut),
	}, func(ok bool) {
		if !ok {
			return
		}
		s.pause()
		s.remember()
		c := &s.project.Clips[s.selected]
		c.Title = strings.TrimSpace(title.Text)
		c.FadeIn = math.Min(c.Duration(), fadeValue(fadeIn.Selected))
		c.FadeOut = math.Min(c.Duration(), fadeValue(fadeOut.Selected))
		s.changed("Updated clip appearance (visible in exported movie)")
	}, s.window).Show()
}

func (s *Studio) insertClip() {
	if s.libraryChoice < 0 || s.libraryChoice >= len(s.library) {
		dialog.ShowInformation("Choose a video", "Import a file and select it in the media library first.", s.window)
		return
	}
	s.insertAt(s.playhead, s.library[s.libraryChoice])
}

func (s *Studio) insertAt(at float64, c editor.Clip) {
	s.pause()
	c.ID = s.id()
	at = s.snapTime(at)
	before := s.project.Clone()
	index, err := s.project.InsertAtTime(at, c, s.id())
	if err != nil {
		s.problem(err)
		return
	}
	s.undo = append(s.undo, before)
	s.redo = nil
	s.selected = index
	s.seek(s.project.Start(index))
	s.changed("Inserted " + c.Name + " at playhead")
}

func (s *Studio) split() {
	s.pause()
	before := s.project.Clone()
	idx, err := s.project.SplitAt(s.playhead, s.id())
	if err != nil {
		s.problem(err)
		return
	}
	s.undo = append(s.undo, before)
	s.redo = nil
	s.selected = idx
	s.changed("Split clip at " + timecode(s.playhead))
}

func (s *Studio) deleteSelected() {
	if s.selected < 0 || s.selected >= len(s.project.Clips) {
		return
	}
	s.pause()
	s.remember()
	_ = s.project.Remove(s.selected)
	if s.selected >= len(s.project.Clips) {
		s.selected = len(s.project.Clips) - 1
	}
	s.changed("Clip removed (source file preserved)")
}

func (s *Studio) remember() {
	s.undo = append(s.undo, s.project.Clone())
	if len(s.undo) > 80 {
		s.undo = s.undo[len(s.undo)-80:]
	}
	s.redo = nil
}
func (s *Studio) undoEdit() {
	if len(s.undo) == 0 {
		return
	}
	s.pause()
	s.redo = append(s.redo, s.project.Clone())
	s.project = s.undo[len(s.undo)-1]
	s.undo = s.undo[:len(s.undo)-1]
	s.selected = -1
	s.changed("Undo")
}
func (s *Studio) redoEdit() {
	if len(s.redo) == 0 {
		return
	}
	s.pause()
	s.undo = append(s.undo, s.project.Clone())
	s.project = s.redo[len(s.redo)-1]
	s.redo = s.redo[:len(s.redo)-1]
	s.selected = -1
	s.changed("Redo")
}
func (s *Studio) changed(status string) {
	s.stopPlayer()
	total := s.project.Duration()
	if s.playhead > total {
		s.playhead = total
	}
	s.info.SetText(status)
	s.clock.SetText(timecode(s.playhead) + " / " + timecode(total))
	if s.selected >= 0 && s.selected < len(s.project.Clips) {
		c := s.project.Clips[s.selected]
		s.selectedInfo.SetText(fmt.Sprintf("%s · Source %s – %s · Clip %s", c.Name, timecode(c.SourceStart), timecode(c.SourceEnd), timecode(c.Duration())))
	} else {
		s.selectedInfo.SetText("No clip selected")
	}
	s.timeline.Refresh()
	s.timelineScroll.Refresh()
	s.requestPreview()
}
func (s *Studio) seek(t float64) {
	s.playhead = math.Max(0, math.Min(t, s.project.Duration()))
	s.clock.SetText(timecode(s.playhead) + " / " + timecode(s.project.Duration()))
	s.timeline.Refresh()
	if s.player != nil && !s.playing {
		player := s.player
		at := s.playhead
		go func() { _ = player.Seek(context.Background(), at) }()
	}
	s.requestPreview()
}

func (s *Studio) requestPreview() {
	if s.playing {
		return
	} // mpv owns continuous playback; thumbnail is for still seeking.
	i, src, ok := s.project.At(s.playhead)
	if !ok {
		s.previewSeq++
		if s.previewCancel != nil {
			s.previewCancel()
		}
		s.lastFrameKey = ""
		s.preview.Image = image.NewRGBA(image.Rect(0, 0, 16, 9))
		s.preview.Refresh()
		return
	}
	c := s.project.Clips[i]
	// Still-image scrubbing samples at 4 FPS; actual playback uses mpv.
	quantized := math.Floor(src*4) / 4
	key := fmt.Sprintf("%s/%.2f", c.Path, quantized)
	if key == s.lastFrameKey {
		return
	}
	s.lastFrameKey = key
	s.previewSeq++
	seq := s.previewSeq
	if s.previewCancel != nil {
		s.previewCancel()
	}
	if img, ok := s.frames[key]; ok {
		s.preview.Image = img
		s.preview.Refresh()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s.previewCancel = cancel
	go func() {
		defer cancel()
		img, err := media.Frame(ctx, c.Path, quantized)
		if err != nil {
			return
		}
		fyne.Do(func() {
			if seq != s.previewSeq {
				return
			}
			if len(s.frames) >= 32 {
				s.frames = make(map[string]image.Image)
			}
			s.frames[key] = img
			s.preview.Image = img
			s.preview.Refresh()
		})
	}()
}

// stopVideoStream terminates the bounded FFmpeg frame producer promptly.
func (s *Studio) stopVideoStream() {
	if s.previewVideoCancel != nil {
		s.previewVideoCancel()
		s.previewVideoCancel = nil
	}
	s.previewVideoFrames = nil
	s.previewVideoNext = nil
	s.previewVideoClip = -1
	s.previewVideoStart = 0
	s.previewVideoSourceStart = 0
}

// advanceVideoPreview runs on the Fyne thread. mpv's position remains the
// authority; the frame producer is allowed to decode ahead but never dictates
// playback speed. Cut changes restart decoding at the edited source in-point.
func (s *Studio) advanceVideoPreview(at float64) {
	if !s.embeddedPreview || !s.playing {
		return
	}
	idx, source, ok := s.project.At(at)
	if !ok {
		return
	}
	if idx != s.previewVideoClip || at+0.1 < s.previewVideoStart {
		s.stopVideoStream()
		clip := s.project.Clips[idx]
		ctx, cancel := context.WithCancel(context.Background())
		frames, err := media.StreamFrames(ctx, clip.Path, source, clip.SourceEnd-source, 15)
		if err != nil {
			cancel()
			s.info.SetText("Video preview unavailable: " + err.Error())
			return
		}
		s.previewVideoCancel = cancel
		s.previewVideoFrames = frames
		s.previewVideoClip = idx
		s.previewVideoStart = at
		s.previewVideoSourceStart = source
	}
	// Catch up with mpv's audio clock, drawing only the newest due frame.
	// Refresh once per poll to avoid flooding Fyne with GPU texture uploads
	// if the decoder has several overdue packets ready at once.
	var displayed image.Image
	defer func() {
		if displayed != nil {
			s.preview.Image = displayed
			s.preview.Refresh()
		}
	}()
	for {
		if s.previewVideoNext == nil {
			select {
			case packet, ok := <-s.previewVideoFrames:
				if !ok {
					s.previewVideoFrames = nil
					return
				}
				s.previewVideoNext = &packet
			default:
				return
			}
		}
		due := s.previewVideoStart + (s.previewVideoNext.SourceTime - s.previewVideoSourceStart)
		if due > at+0.045 {
			return
		}
		displayed = s.previewVideoNext.Image
		s.previewVideoNext = nil
	}
}

func (s *Studio) togglePlayback() {
	if s.playing {
		s.pause()
		return
	}
	if len(s.project.Clips) == 0 {
		return
	}
	if s.playhead >= s.project.Duration() {
		s.seek(0)
	}
	s.playing = true
	s.previewSeq++
	if s.previewCancel != nil {
		s.previewCancel()
	}
	s.stopVideoStream()
	s.playButton.SetText("Pause")
	s.playButton.SetIcon(theme.MediaPauseIcon())
	generation := s.playerGeneration
	if s.player != nil {
		player := s.player
		go func() {
			if err := player.Pause(context.Background(), false); err != nil {
				fyne.Do(func() {
					if s.player == player {
						s.stopPlayer()
						s.problem(err)
					}
				})
			}
		}()
		s.pollPlayer(player, generation)
		return
	}
	snapshot := s.project.Clone()
	at := s.playhead
	s.info.SetText("Starting synchronized preview…")
	go func() {
		var player *media.Player
		var err error
		if s.embeddedPreview {
			player, err = media.StartAudioPlayer(context.Background(), snapshot, at)
		} else {
			player, err = media.StartPlayer(context.Background(), snapshot, at)
		}
		fyne.Do(func() {
			if generation != s.playerGeneration || !s.playing {
				if player != nil {
					player.Close()
				}
				return
			}
			if err != nil {
				s.pause()
				s.problem(err)
				return
			}
			s.player = player
			if s.embeddedPreview {
				s.info.SetText("Playing in Fyne · mpv audio clock (beta)")
			} else {
				s.info.SetText("Playing in mpv window · timeline follows mpv's clock")
			}
			s.pollPlayer(player, generation)
		})
	}()
}
func (s *Studio) pollPlayer(player *media.Player, generation uint64) {
	stop := make(chan struct{})
	s.playbackStop = stop
	go func() {
		ticker := time.NewTicker(65 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
			}
			ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
			pos, err := player.Position(ctx)
			cancel()
			fyne.Do(func() {
				if !s.playing || s.player != player || generation != s.playerGeneration {
					return
				}
				if err != nil {
					s.stopPlayer()
					s.info.SetText("Preview player stopped: " + err.Error())
					return
				}
				if pos >= s.project.Duration()-0.025 {
					s.pause()
					s.seek(s.project.Duration())
					return
				}
				s.playhead = math.Max(0, math.Min(pos, s.project.Duration()))
				s.clock.SetText(timecode(s.playhead) + " / " + timecode(s.project.Duration()))
				s.timeline.Refresh()
				s.advanceVideoPreview(s.playhead)
			})
		}
	}()
}
func (s *Studio) pause() {
	if !s.playing {
		return
	}
	s.playing = false
	s.stopVideoStream()
	if s.player == nil {
		s.playerGeneration++
	} // stale asynchronous startup cannot adopt a player after pause
	if s.playbackStop != nil {
		close(s.playbackStop)
		s.playbackStop = nil
	}
	s.playButton.SetText("Play")
	s.playButton.SetIcon(theme.MediaPlayIcon())
	if s.player != nil {
		player := s.player
		go func() { _ = player.Pause(context.Background(), true) }()
	}
	s.requestPreview()
}
func (s *Studio) stopPlayer() {
	s.stopVideoStream()
	s.playerGeneration++
	s.pause()
	if s.player != nil {
		s.player.Close()
		s.player = nil
	}
}

func (s *Studio) problem(err error) {
	if err == nil {
		return
	}
	s.info.SetText("Error: " + err.Error())
	dialog.ShowError(err, s.window)
}

func (s *Studio) newProject() {
	dialog.ShowConfirm("New project?", "Clear this timeline? Original media files will not be touched.", func(yes bool) {
		if !yes {
			return
		}
		s.stopPlayer()
		s.project = editor.NewProject()
		s.projectPath = ""
		s.selected = -1
		s.undo = nil
		s.redo = nil
		s.seek(0)
		s.changed("New project")
	}, s.window)
}
func (s *Studio) saveProject() {
	if s.projectPath != "" {
		s.writeProject(s.projectPath)
		return
	}
	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			s.problem(err)
			return
		}
		if w == nil {
			return
		}
		path := w.URI().Path()
		_ = w.Close()
		if !strings.HasSuffix(strings.ToLower(path), ".velvetcut") {
			path += ".velvetcut"
		}
		s.writeProject(path)
	}, s.window)
	d.SetFileName("Untitled.velvetcut")
	d.Show()
}
func (s *Studio) writeProject(path string) {
	data, err := json.MarshalIndent(s.project, "", "  ")
	if err != nil {
		s.problem(err)
		return
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		s.problem(err)
		return
	}
	s.projectPath = path
	s.info.SetText("Saved project · " + filepath.Base(path))
}
func (s *Studio) openProject() {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			s.problem(err)
			return
		}
		if r == nil {
			return
		}
		path := r.URI().Path()
		_ = r.Close()
		data, err := os.ReadFile(path)
		if err != nil {
			s.problem(err)
			return
		}
		var p editor.Project
		if err = json.Unmarshal(data, &p); err != nil {
			s.problem(err)
			return
		}
		if err = p.Validate(); err != nil {
			s.problem(err)
			return
		}
		for _, c := range p.Clips {
			if _, err := os.Stat(c.Path); err != nil {
				s.problem(fmt.Errorf("missing media %q: %w", c.Path, err))
				return
			}
		}
		s.stopPlayer()
		s.project = p
		s.projectPath = path
		s.undo = nil
		s.redo = nil
		s.selected = -1
		s.library = nil
		s.libraryChoice = -1
		added := map[string]bool{}
		for _, c := range p.Clips {
			if !added[c.Path] {
				s.library = append(s.library, c)
				s.makeThumbnail(c)
				s.makeWaveform(c)
				added[c.Path] = true
			}
		}
		s.mediaList.Refresh()
		s.seek(0)
		s.changed("Opened " + filepath.Base(path))
	}, s.window)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".velvetcut"}))
	d.Show()
}
func (s *Studio) exportMovie() {
	if len(s.project.Clips) == 0 {
		dialog.ShowInformation("Nothing to export", "Insert some clips first.", s.window)
		return
	}
	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			s.problem(err)
			return
		}
		if w == nil {
			return
		}
		path := w.URI().Path()
		_ = w.Close()
		if !strings.HasSuffix(strings.ToLower(path), ".mp4") {
			path += ".mp4"
		}
		ctx, cancel := context.WithCancel(context.Background())
		progress := widget.NewProgressBarInfinite()
		cancelButton := widget.NewButton("Cancel export", cancel)
		popup := widget.NewModalPopUp(container.NewVBox(
			widget.NewLabelWithStyle("Rendering MP4", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Encoding H.264 video + AAC audio (1280 × 720, 30 fps)"),
			progress, cancelButton,
		), s.window.Canvas())
		popup.Show()
		snapshot := s.project.Clone()
		s.info.SetText("Exporting movie…")
		go func() {
			err := media.Export(ctx, snapshot, path)
			fyne.Do(func() {
				popup.Hide()
				if err != nil {
					s.problem(err)
					return
				}
				s.info.SetText("Exported " + filepath.Base(path))
				dialog.ShowInformation("Export complete", "Movie saved to:\n"+path, s.window)
			})
		}()
	}, s.window)
	d.SetFileName("VelvetCut-movie.mp4")
	d.Show()
}
