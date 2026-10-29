package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/SeraphinaDX/VelvetCut/editor"
	"github.com/SeraphinaDX/VelvetCut/media"
)

func isAudioPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3", ".wav", ".m4a", ".aac", ".flac", ".ogg", ".opus", ".aiff", ".aif":
		return true
	}
	return false
}

func (s *Studio) importAudio() {
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
		s.importAudioPath(path, s.playhead)
	}, s.window)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".mp3", ".wav", ".m4a", ".aac", ".flac", ".ogg", ".opus", ".aiff", ".aif"}))
	d.Show()
}

func (s *Studio) importAudioPath(path string, at float64) {
	if !isAudioPath(path) {
		s.info.SetText("Unsupported audio format: " + filepath.Base(path))
		return
	}
	s.info.SetText("Inspecting audio " + filepath.Base(path) + "…")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		length, err := media.ProbeAudio(ctx, path)
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
			clip := editor.AudioClip{ID: s.id(), Path: abs, Name: filepath.Base(abs), Start: math.Max(0, at), SourceStart: 0, SourceEnd: length, SourceDuration: length, Volume: 1}
			s.pause()
			s.remember()
			if err := s.project.AddAudio(clip); err != nil {
				s.problem(err)
				return
			}
			s.selected = -1
			s.selectedAudio = len(s.project.AudioClips) - 1
			s.makeWaveform(editor.Clip{Path: abs, HasAudio: true, SourceDuration: length})
			s.changed("Audio added: " + clip.Name + " · drag to move or trim")
		})
	}()
}

func (s *Studio) detachAudio() {
	if s.selected < 0 || s.selected >= len(s.project.Clips) {
		dialog.ShowInformation("Select a video clip", "Click a video clip to detach its original audio.", s.window)
		return
	}
	s.pause()
	before := s.project.Clone()
	if err := s.project.DetachAudio(s.selected, s.id()); err != nil {
		s.problem(err)
		return
	}
	s.undo = append(s.undo, before)
	s.redo = nil
	s.selected = -1
	s.selectedAudio = len(s.project.AudioClips) - 1
	s.changed("Detached video sound onto AUDIO 1")
}

// editAudio controls either an independent audio segment or a video's original
// sound. A nil video AudioVolume means 100% for old saved projects.
func (s *Studio) editAudio() {
	isExtra := s.selectedAudio >= 0 && s.selectedAudio < len(s.project.AudioClips)
	isVideo := s.selected >= 0 && s.selected < len(s.project.Clips)
	if !isExtra && !isVideo {
		dialog.ShowInformation("Select a clip", "Select an audio or video clip to edit its sound.", s.window)
		return
	}
	name := ""
	initialVolume := 1.0
	muted := false
	fadeIn, fadeOut := 0.0, 0.0
	if isExtra {
		a := s.project.AudioClips[s.selectedAudio]
		name = a.Name
		initialVolume = a.Volume
		muted = a.Muted
		fadeIn = a.FadeIn
		fadeOut = a.FadeOut
	} else {
		c := s.project.Clips[s.selected]
		name = c.Name
		initialVolume = c.ClipAudioVolume()
		muted = c.AudioMuted
		if !c.HasAudio {
			dialog.ShowInformation("Silent video", "This video has no original sound.", s.window)
			return
		}
	}
	volume := widget.NewSlider(0, 200)
	volume.Step = 5
	volume.SetValue(initialVolume * 100)
	level := widget.NewLabel(fmt.Sprintf("%.0f%%", initialVolume*100))
	volume.OnChanged = func(v float64) { level.SetText(fmt.Sprintf("%.0f%%", v)) }
	mute := widget.NewCheck("Mute sound", nil)
	mute.SetChecked(muted)
	fields := []*widget.FormItem{widget.NewFormItem("Volume", volume), widget.NewFormItem("Level", level), widget.NewFormItem("Mute", mute)}
	var in, out *widget.Entry
	if isExtra {
		in = widget.NewEntry()
		in.SetText(fmt.Sprintf("%.2f", fadeIn))
		out = widget.NewEntry()
		out.SetText(fmt.Sprintf("%.2f", fadeOut))
		fields = append(fields, widget.NewFormItem("Audio fade in (s)", in), widget.NewFormItem("Audio fade out (s)", out))
	}
	targetAudio := s.selectedAudio
	targetVideo := s.selected
	dialog.NewForm("Sound · "+name, "Apply", "Cancel", fields, func(ok bool) {
		if !ok {
			return
		}
		v := volume.Value / 100
		fadeStart, fadeEnd := 0.0, 0.0
		if isExtra {
			var err error
			fadeStart, err = strconv.ParseFloat(strings.TrimSpace(in.Text), 64)
			if err != nil {
				s.problem(fmt.Errorf("fade in: %w", err))
				return
			}
			fadeEnd, err = strconv.ParseFloat(strings.TrimSpace(out.Text), 64)
			if err != nil {
				s.problem(fmt.Errorf("fade out: %w", err))
				return
			}
			a := s.project.AudioClips[targetAudio]
			a.Volume = v
			a.Muted = mute.Checked
			a.FadeIn = fadeStart
			a.FadeOut = fadeEnd
			if !a.Valid() {
				s.problem(fmt.Errorf("volume must be 0–200%% and fades must fit within clip duration"))
				return
			}
			s.pause()
			s.remember()
			s.project.AudioClips[targetAudio] = a
		} else {
			s.pause()
			s.remember()
			c := &s.project.Clips[targetVideo]
			c.AudioVolume = &v
			c.AudioMuted = mute.Checked
		}
		s.changed("Updated audio settings · next Play renders mixed soundtrack")
	}, s.window).Show()
}
