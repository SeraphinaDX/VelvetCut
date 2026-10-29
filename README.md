# VelvetCut 0.5.0 (beta)

**VelvetCut** is a GPL-3.0-or-later, non-destructive iMovie-inspired video editor written in Go and [Fyne](https://fyne.io/). It uses FFmpeg/FFprobe to decode/export clips and mpv to keep live playback synchronized.

## Audio editing in 0.5.0

- **Three independent audio lanes** under the video timeline. Add music, voiceover recordings, or sound effects via **Add music / SFX** (MP3, WAV, M4A, AAC, FLAC, Ogg, Opus, AIFF) or drop an audio file into the window.
- **Drag audio clips** to a new start time, or drag either edge to trim. Audio edits do not ripple video or affect other audio clips. Audio clips may overlap; different lanes help keep them visible.
- **Select audio**, click **Volume / fades** to set 0–200% volume, mute, fade in/out duration in seconds, or choose Audio 1/2/3. Video clips also have their own volume/mute controls in this dialog.
- **Detach audio** from a selected video clip. Original video sound is muted and a separately editable sound clip is added at the original timeline position. Video footage is unchanged.
- Final MP4 output **mixes all audio lanes** with the video's original soundtrack. Fades, volume and mute also appear in the **in-editor rendered preview**. The preview proxy is regenerated after audio or transition edits.
- Projects save audio sources as references, not copies; existing `.velvetcut` projects open unchanged. Original video/audio files are never overwritten.

## Existing video features

- Import video (MP4/MOV/MKV/WebM/AVI/M4V), insert at playhead, split, reorder, trim either edge, drag/drop files.
- Accurate FFmpeg-based cross-dissolves and fade-through-black with audio crossfades.
- Clip titles, fades to/from black, video thumbnail/waveform timeline, snapping and keyboard shortcuts.
- Undo/redo, save/open JSON `.velvetcut` projects; export H.264/AAC MP4 (1280×720 at 30 fps).
- In-app video playback via FFmpeg frame streaming (15 fps / 640×360) and mpv audio. The separate mpv playback window remains available as a fallback.

## Build and run (CachyOS / Arch Linux)

```sh
sudo pacman -S --needed go ffmpeg mpv base-devel mesa libglvnd libx11 libxcursor libxrandr libxinerama libxi

go mod tidy
go run .
```

Go 1.22+ and Fyne v2.8.1 are required. Test the editing/media logic independently:

```sh
go test ./editor ./media
```

## Usage example

1. Import and insert two video clips, then apply a cross-dissolve if desired.
2. Move the playhead to 00:03, click **Add music / SFX** and select a song.
3. Drag its block on AUDIO 1 to align the music with the video, or drag its edge to shorten it.
4. Select it, open **Volume / fades**, set **Volume** to 35%, **Fade in** to 1, **Fade out** to 2. Move it to AUDIO 2 if you want room for voiceover on AUDIO 1.
5. Click a video clip and choose **Detach audio** to edit its soundtrack independently.
6. Press Play; VelvetCut renders a temporary proxy containing music and transitions, then plays it inside the editor. Export MP4 to create the movie.

## Known limitations

- Edited audio or transitions require a temporary FFmpeg proxy render before preview playback; it is cached until an edit changes the project.
- Video thumbnails still scrub at 4 fps; animated in-editor playback targets 15 fps. Effects playback is a preview proxy, not real-time GPU compositing.
- Audio clips beyond the video end are trimmed during export; clips can overlap, and total mixed loudness can clip if raised too high (use lower gains to avoid distortion).
- Three fixed visual audio lanes are provided; drag moves clips horizontally, and **Volume / fades** changes lanes. Auto-ducking, recording, stereo panning and per-track mixer controls are not implemented yet.
- Source paths are absolute and require files to remain at their original locations. Windows/macOS playback is not tested.
- Fyne desktop preview requires hands-on testing on Linux; the package-level FFmpeg tests do not launch the window.

## Source

- `editor/project.go`, `editor/audio.go`, `editor/transition.go`: edit model and timeline timing.
- `audio_ui.go`, `main.go`, `timeline.go`: audio tools, controls and Fyne timeline.
- `media/audio.go`, `media/ffmpeg.go`: audio metadata, export/mix/proxy.

License: **GPL-3.0-or-later**. See `COPYING`.
