# Changelog

## 0.4.0 (beta) — 2026-10-25

- Add overlapping cross-dissolves and fade-through-black at clip boundaries with synchronized audio crossfades.
- Persist per-clip outgoing transition metadata and update non-destructive timeline duration, cuts and snapping.
- Add a Transition dialog plus draggable purple timeline transition regions with length validation.
- Cache low-resolution, effects-accurate MP4 preview proxies when a movie contains transitions, for Fyne or classic mpv.
- Limit excessive waveform drawing and calculate clip positions in a linear timeline pass.
- Preserve the 0.3.0 one-frame preview fix; add real FFmpeg transition and model tests.

**Beta:** Building a transition preview renders the movie in the background before playback. Full Fyne GUI testing on Linux is still needed.


## 0.3.0 beta preview hotfix — 2026-10-25

- Fix the in-editor preview freezing after one frame while audio and playhead continued. The FFmpeg streaming backend now reads fixed-size RGBA frames instead of concatenated MJPEG images, where the JPEG decoder could consume bytes belonging to the next frame.
- Limit frame buffering to two decoded images; catch up to the mpv clock with at most one Fyne canvas refresh per poll.
- Add an FFmpeg regression test for sustained frame progression and aspect-ratio-safe portrait videos.


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
