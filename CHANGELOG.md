# Changelog

## 0.5.0 (beta) — 2026-10-29

- Add three independent, individually movable/trimmed audio lanes with waveform previews.
- Import audio files (MP3, WAV, M4A, AAC, FLAC, Ogg/Opus, AIFF), or drop them into VelvetCut.
- Add per-audio-clip volume (0–200%), mute, fades and configurable lane.
- Add per-video-clip volume and mute, including a detach-audio operation that creates an independent clip and mutes the original.
- Mix independent sound layers with existing video sound and transition crossfades using FFmpeg; match in-editor preview with export via cached audio/effects proxy.
- Retain compatibility with older version-1 `.velvetcut` projects.
- Add independent audio model and real FFmpeg export regression tests.


## 0.4.0 (beta) — 2026-10-25

- Add genuine overlapping cross-dissolves and fade-through-black between neighboring clips, with matched audio crossfades.
- Add per-clip outgoing transition data, validation of available source footage, recalculated project duration, edit-time normalization and backward-compatible project loading.
- Add purple resizeable transition regions on the timeline and the Transition ⇄ dialog.
- Render and cache temporary effects-accurate preview MP4s when the project has transitions; use the proxy for both synchronized Fyne and classic mpv playback.
- Cap waveform rendering and make redraw clip placement linear for long timelines.
- Preserve the fixed-size raw RGBA frame-stream fix from the 0.3.0 playback hotfix.
- Add model and FFmpeg integration tests for real transitions, their duration, and audio.

**Beta limitations:** Transition previews require an upfront FFmpeg proxy render that can be slow on large projects, and live GUI/mpv desktop playback still needs testing on CachyOS.

## 0.3.0 (beta) — 2026-10-25

- Add experimental in-editor 15 fps video preview, using incremental FFmpeg MJPEG frame decoding and mpv audio clock. Preserve classic separate-window mpv playback as fallback.
- Add desktop file drop-to-import, and drop-to-insert when the file is released over the video timeline.
- Add in-app media library clip drag-and-drop to ripple-insert onto the timeline.
- Add snapping of playhead seeking and clip insert points to existing cut boundaries with toggleable 8-pixel tolerance.
- Add editing/transport keyboard shortcuts (Space, S, Delete, arrows, Home/End, Ctrl+Z/Y/S/O).
- Fix the Go module path to `github.com/SeraphinaDX/VelvetCut`.
- Add tests covering snapping and bounded FFmpeg streamed video frames.

**Testing note:** Go editing/media tests pass locally. Fyne desktop and mpv A/V playback require validation on Linux; in-editor preview is experimental and is not effects-accurate.

## 0.2.0 — 2026-10-25

- Add mpv synchronized external video/audio preview with virtual EDL and local IPC timeline position.
- Add audio waveforms aligned to trimmed source clips.
- Add per-clip titles and black fades (with matching audio fades) to exported MP4s.
- Add validation for fade parameters and integration tests for export/title/fade/waveform/EDL.
- Maintain project v1 compatibility.

## 0.1.0 — 2026-10-25

- Initial non-destructive Fyne timeline: insert, split, trim, reorder, thumbnail scrub preview, undo/redo, project save/open, and MP4 export.
