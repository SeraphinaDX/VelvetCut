# VelvetCut 0.6.0 beta

**VelvetCut** is a GPL-3.0-or-later video editor written in Go + Fyne, inspired by iMovie. FFmpeg handles media decoding and rendering; mpv supplies a synchronized playback clock.

## 0.6.0: Clip inspector, titles, and visual effects

Select a **video** clip to open its docked **Clip Inspector** on the right. Three tabs provide non-destructive settings:

- **Title:** edit text, choose Sans/Serif/Mono, white/yellow/cyan/pink/black, title location, and a fade, slide, or zoom animation. Set font size and title duration.
- **Colour:** brightness, contrast, saturation, and warm/cool temperature sliders.
- **Effects:** none, black-and-white, sepia, or vignette; Fit, Fill, Square, or 4:3 center cropping; 0°/90°/180°/270° clockwise rotation; center-crop zoom.

Press **Apply to selected clip** to store the edit as **one undoable operation**. Paused-frame scrubbing refreshes with the same visual filters that final MP4 export uses. Click **Play** for an **effects-accurate** in-editor video preview; VelvetCut first renders a lower-resolution proxy (cached until the project changes), so large projects may take longer to start. The proxy preview and final MP4 use the same FFmpeg visual filters, at different resolutions and frame rates. Existing transitions and independent audio tracks are preserved.

**Note:** The inspector controls are staged until you press Apply, rather than rebuilding a full video proxy on every slider movement. Titles are overlays on existing clips, not separate movable title tracks. Cropping is centered rather than free-form mouse cropping. Color temperature is a red/blue balance approximation.

## Existing features

- Media library with timeline dragging and desktop file drops.
- Video clip insertion, splitting, trimming, reordering, zoom, snapping, shortcuts, undo/redo.
- Cross-dissolves, fade-through-black transitions with audio crossfade and draggable durations.
- Three independent audio lanes: music/sound effects, volume, fades, trimming, repositioning and detached clip soundtracks.
- H.264/AAC MP4 export with synchronized mpv + FFmpeg in-editor preview. Fallback external mpv playback.
- Non-destructive `.velvetcut` JSON projects; 0.1.x–0.5.x project files remain compatible.

## Install and run (CachyOS / Arch)

```sh
sudo pacman -S --needed go ffmpeg mpv base-devel mesa libglvnd libx11 libxcursor libxrandr libxinerama libxi
go mod tidy
go run .
```

Run model and real FFmpeg rendering tests:

```sh
go test ./editor ./media
```

The new tests render the animated title presets, cropping and rotations, all effects, audio-bearing MP4s, and paused styled frames. Desktop Fyne UI testing still needs a Linux machine with working OpenGL dependencies.

## Limitations

- The in-editor playback path displays a 640×360 / 15 fps preview. FFmpeg proxies are slower to produce for longer timelines, but cached until edits occur.
- Output defaults to 1280×720 at 30 fps. Crop aspect modes letterbox into this canvas.
- Saved projects still reference absolute media paths. Moving media can break links.
- No keyframe animation editor, freehand crop, or standalone title tracks yet.

Licensed GPL-3.0-or-later; see `COPYING`.
