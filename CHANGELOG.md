# Changelog

## 0.8.0 (beta) — 2026-10-29

- Add normalized per-clip video camera keyframes for interpolated pan and zoom (Ken Burns-style camera movement).
- Add a Camera tab with focus/zoom sliders, keyframe creation and navigation, deletion, and a draggable framing guide in the preview.
- Show cyan video-motion markers alongside existing gold title keyframes.
- Apply identical FFmpeg zoompan expressions to styled stills, composited playback proxies, and final MP4 export.
- Preserve camera keyframes through split, trim, save, undo/redo and backward-compatible project JSON.
- Add regression tests for interpolation, split/trim, validation, JSON persistence, and actual camera-motion preview/export.

## 0.7.0 (beta) — 2026-10-29

- Add normalized title position, scale and opacity keyframes with linear interpolation.
- Drag a selected title's outline over the movie preview to create one undoable position keyframe.
- Add an Inspector Motion tab with scale/opacity controls, add/update/delete/clear and previous/next navigation.
- Show gold keyframe markers on the video timeline.
- Preserve title animation during split/trim; deep-copy keyframes into undo history; save/load keyframes in existing JSON project v1 format.
- Share per-frame FFmpeg `drawtext` expressions across still preview, playback proxy and final MP4 export; keep legacy title presets for unkeyframed clips.
- Add model and FFmpeg integration tests for interpolation, split/trim, saving, motion rendering and backward compatibility.

## 0.6.0 (beta)

- Add docked clip inspector with animated title presets, fonts, colours, layout, brightness, contrast, saturation, temperature, colour effects, cropping, rotation, and image zoom.
- Use matching FFmpeg visual filters for paused stills, rendered preview proxy and export.

## 0.5.0 (beta)

- Add independent audio lanes, music clips, volume/mute, audio fade, trim, drag, detached audio and export mixing.

## 0.4.0 (beta)

- Add real overlapping cross-dissolve and fade-through-black transitions, audio crossfades, timeline transition handles and effects-accurate preview.

## 0.3.0 (beta)

- Add embedded FFmpeg preview, library drag-and-drop, snapping, shortcuts and module path cleanup.

## 0.2.0

- Add audio waveforms, titles, clip fades and synchronized mpv playback.

## 0.1.0

- Initial clip import, split, trim, reorder, project save and MP4 export.
