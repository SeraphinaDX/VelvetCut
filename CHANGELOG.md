# Changelog

## 0.2.0 — 2026-10-25

- Add separate-window mpv synchronized video/audio preview, virtual EDL timeline and local IPC playhead updates.
- Add audio waveforms with source-time-aware trimming.
- Add titles (first 3 seconds) and clip fade-from/to-black plus audio fade for MP4 exports.
- Add validations for fade durations, transform fade metadata on trim and split.
- Add integration tests for title and fades, waveform and EDL escaping.
- Maintain compatibility with the original `.velvetcut` v1 project format.

**Known limitations:** Preview window is separate; mpv preview does not composite titles/fades, and mixed audio/no-audio sources may not preview consistently. Export handles the latter. True cross-dissolves and multitrack editing are not yet implemented.

## 0.1.0 — 2026-10-25

- Initial non-destructive Fyne timeline with insertion, split, edge-trim, reordering, still-image preview, undo/redo, save/load, MP4 export.
