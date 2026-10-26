# VelvetCut 0.4.0 (beta)

**VelvetCut** is a GPL-3.0-or-later non-destructive video editor inspired by iMovie, written in Go using Fyne. FFmpeg handles media analysis, streaming frames, and MP4 encoding; mpv provides a synchronized audio clock.

## Preview freeze hotfix

The 0.3.0 beta in-editor video sometimes froze after its first frame while audio kept playing. FFmpeg now streams fixed-size RGBA frames rather than MJPEG frames that were incorrectly decoded across image boundaries. Run `go test ./media` for the continuous-frame regression test.

## What's new in 0.4.0 (beta)

- **Real clip-to-clip transitions:** Select a clip and choose **Transition ⇄** to apply **Cross dissolve** or **Fade through black** into the following clip. Enter the duration in seconds or drag the purple overlap region on the timeline to resize it.
- **Matching audio:** Both video and audio crossfade across the overlap. A 0.75-second transition overlaps 0.75 seconds of outgoing/incoming video, shortening the finished movie by that amount.
- **Effects-accurate playback:** On Play, when transitions exist, FFmpeg generates a temporary 640×360 / 15fps preview MP4 using the same video/audio graph as MP4 export. The Fyne preview and classic mpv-window mode then play the proxy. Proxies are reused while the project is unchanged.
- **Edit safety:** Transition validation avoids impossible overlaps. Trimming may shorten adjacent transitions; moving clips clears affected transition settings rather than silently blending unrelated footage.
- **Performance:** Timeline clip placement is linear, and the amount of waveform strokes per very long clip is capped.
- **Backward compatibility:** Existing .velvetcut files remain readable; new transition fields are optional and source videos are untouched.

**Try it:** Insert at least two clips, click the first, choose **Transition ⇄**, select Cross dissolve and enter 0.75 seconds. Press Play. The first playback will render a preview proxy before playing, and a later edit will regenerate it. **Pause** cancels a pending render. Final **Export MP4** renders in 1280×720 at 30fps.

**Known limitations:** Long projects may take significant time and temporary disk space to render a transition preview. Full-speed in-editor compositing is not implemented; effects-accurate playback uses a low-resolution render first. The desktop Fyne/mpv integration needs testing on Linux. Existing fades/titles are visible in preview proxies only when the movie contains a transition; plain-cut EDL preview still omits those effects.

## What's new in 0.3.0

- **In-editor preview (beta):** Fyne displays continuously decoded frames (up to **15 fps**, 640×360) while mpv plays audio without opening a video window. This is an early preview path, **not yet full-frame-rate** or effects-accurate. The playhead follows mpv's actual clock. **Play video in editor (beta)** is on by default.
- **Classic playback fallback:** Uncheck **Play video in editor (beta)** to restore the separate mpv video window used in 0.2.0. This is useful if the embedded preview lags or a source has incompatible media tracks.
- **Drag videos from the file manager** onto VelvetCut: drops inside the visible timeline insert at that time; drops elsewhere add videos to the library.
- **Drag clips from the media library to the timeline** to ripple-insert at the drop position. The original **Insert at playhead** action still works.
- **Magnetic playhead snapping:** the timeline ruler, dragging the playhead and library insert positions snap to nearby cuts (8 logical pixels) with **Snap to cuts** enabled. This does not alter the actual source range during edge trimming.
- **Keyboard shortcuts:** Space = play/pause, S = split at playhead, Delete / Backspace = remove selected clip, Left/Right arrows = step one frame (at 30 fps), Home/End = seek to start/end. Ctrl+Z / Ctrl+Y = undo/redo; Ctrl+S = save; Ctrl+O = open.
- Fix the Go module import path to `github.com/SeraphinaDX/VelvetCut` (formerly the `github.com/example/velvetcut` placeholder).

## Other features

- Import MP4, MOV, MKV, WebM, AVI and M4V videos.
- Non-destructive video clips, split at playhead, drag the in/out edges to trim, drag clips to reorder.
- Timeline thumbnails and audio waveforms, zoom, undo/redo.
- Apply a simple per-clip title or 0.5s / 1s fade in/out (currently visible in final MP4 exports only).
- Save and load `.velvetcut` editable projects; existing 0.1.x and 0.2.x project files remain readable.
- Export an MP4 using H.264 video (1280×720, 30fps) and AAC audio. FFmpeg will produce silent audio tracks for silent clips so that clips can mix successfully during export.

## Requirements

- Go 1.22 or later; Fyne v2.8.1, compiler and Fyne OpenGL dependencies.
- `ffmpeg` and `ffprobe` in PATH; FFmpeg requires `libx264`, AAC, drawtext for title export.
- `mpv` in PATH for both preview modes.

CachyOS / Arch Linux:

```sh
sudo pacman -S --needed go ffmpeg mpv base-devel mesa libglvnd libx11 libxcursor libxrandr libxinerama libxi
```

Build and run:

```sh
go mod tidy
go run .
```

Tests independent of Fyne's GUI dependencies:

```sh
go test ./editor ./media
```

## Important limitations

- The in-editor preview is **experimental**. It currently decodes only up to 15 fps at 640×360, and the actual frame rate depends on your CPU and UI performance. Audio is handled separately by mpv, with the mpv clock governing which decoded frame is displayed; synchronization is approximate.
- In-editor playback is not a full compositor: clip titles and fades appear on **export**, not in live playback. Full-resolution, effects-accurate live compositing is future work.
- mpv's EDL format can struggle with some combinations of silent and audio-containing media. If the in-editor audio-only player has trouble, turn off **Play video in editor (beta)** for classic external mpv video/audio playback.
- Drag-and-drop behavior and live preview require a desktop test. External drop placement in Fyne v2.8.x has a known Windows issue; Linux is the initial target.
- No clip-to-clip dissolves, music tracks, multi-layer video compositing, or automatic proxy generation yet.
- Saved projects reference absolute media paths: moving the originals can break links.

## Source layout

- `main.go`: Fyne editor, shortcuts, OS drops, playback, MP4 export.
- `library.go`: media library row dragging.
- `timeline.go`: timeline drawing, trim/reorder drag handling, snap-to-cut seeks.
- `editor/project.go`, `editor/snap.go`: non-destructive project operations, snapping.
- `media/ffmpeg.go`: metadata, still frames, exporting.
- `media/stream.go`: bounded incremental FFmpeg frame stream.
- `media/player.go`: separate-video and audio-only mpv playback with IPC.

Licensed GPL-3.0-or-later; see `COPYING`.
