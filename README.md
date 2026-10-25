# VelvetCut 0.2.0

**VelvetCut** is a GPL-3.0-or-later, non-destructive, iMovie-inspired video editor built with **Go and Fyne**. A purple-themed Fyne UI controls the clip library, timeline, editing and FFmpeg exporting.

> **Prototype status:** Importing, splitting, trimming, rearranging, saving, exporting, basic audio waveforms, titles, and fades are implemented. The synchronized real-time preview runs in a **separate mpv window**, not embedded in the Fyne editor. This is still an early editor, not an iMovie replacement.

## What's new in 0.2.0

- **Synchronized live video and audio playback:** Launches a separate mpv playback window. mpv handles A/V sync, and the red Fyne timeline playhead follows mpv's actual playback timestamp via private local JSON IPC. Play/pause, timeline seeks, and close are controlled from VelvetCut. Editing invalidates a stale playback session so that the next play starts the new cut.
- **Audio waveform** rendered underneath thumbnails when a clip contains audio. Peaks are computed by FFmpeg at low resolution on import and align with source in/out points.
- **Clip title text:** Select a timeline clip → **Title & fades**. Titles appear as an overlay for the first three seconds of that clip in the exported movie. Unicode text and punctuation are supported; text is loaded from a safe temporary UTF-8 file.
- **Fade in / fade out:** Apply 0.5s or 1s **fade from/to black and matching audio fade** to selected clips. These are **not clip-to-clip dissolves**.
- Reusable mpv EDL timeline generator and additional model/export tests.
- The editor continues to use still frames inside the Fyne pane when paused/scrubbing.

## Original 0.1.0 features

- Import MP4, MOV, MKV, WebM, AVI and M4V video into a media library.
- Insert a chosen clip **at the playhead**, automatically splitting the existing clip when needed.
- Split at the playhead; drag **either edge** of a clip to trim the start/end; drag the middle to reorder.
- Timeline thumbnails, zoom, playhead scrubbing, selecting and deleting clips, undo and redo.
- Save/reopen `*.velvetcut` JSON project files (referencing the original video files).
- Export 1280×720, 30 fps H.264 video and AAC audio to MP4. Clips without audio get a silent track. Exports are cancellable.
- All edits are non-destructive. Source media remains unchanged.

## Dependencies

- Go **1.22+** (Fyne 2.8.1)
- FFmpeg + FFprobe (including `libx264`, AAC, and `drawtext` filters)
- **mpv** for real-time synchronized video/audio preview (not needed for still scrubbing or exporting)
- A C compiler and desktop OpenGL development headers for Fyne

### CachyOS / Arch Linux

```sh
sudo pacman -S --needed go ffmpeg mpv base-devel mesa libglvnd libx11 libxcursor libxrandr libxinerama libxi
```

### Debian / Ubuntu

```sh
sudo apt install golang-go ffmpeg mpv gcc pkg-config libgl1-mesa-dev xorg-dev
```

Check `go version`: the repository requires Go >=1.22.

## Build and launch

```sh
unzip velvetcut-0.2.0-source.zip
cd velvetcut
# Needs network access on first build to download Fyne dependencies.
go mod tidy
go run .
```

Or:

```sh
go build -o velvetcut .
./velvetcut
```

`go.mod` pins Fyne v2.8.1; the Go module is `github.com/example/velvetcut`, a placeholder until a real source repository is chosen.

## Editing walkthrough

1. Import one or more videos into the library.
2. Select a library item, then **Insert at playhead**.
3. Click the top timeline ruler to move the red playhead, then insert or **Split at playhead**.
4. Drag the left grip to change where a clip begins in the source video, or the right grip to change where it ends. Drag the middle to rearrange clips.
5. Click any clip and choose **Title & fades**. Set the title (optional) and fade in/out durations.
6. Press **Play** to open synchronized playback in **mpv**. The editor's red playhead follows it. Press **Pause** to pause the movie. The Fyne preview pane is a still-image scrubber, not the live mpv player.
7. **Save** your editable project and **Export MP4** for a finished video.

The source paths in a saved project are absolute. Moving files can break references. `*.velvetcut` v1 project files from VelvetCut 0.1.0 remain readable. Title and fade parameters are optional additive JSON properties on each clip.

## Important limitations

- Playback opens in a **separate mpv window** because native full-speed video embedding in Fyne would require substantially more plumbing, possibly `libmpv`/native window handles. The editor's in-app image pane is **silent, still-image scrubbing** (4 fps sampling), not the live stream.
- Title and fade effects appear in **MP4 export**, **not live mpv playback**; the live playback shows the plain source cuts/trims. This is intentional and should be addressed by a composited-preview renderer in a future release.
- mpv's EDL can struggle with heterogeneous media tracks (for example mixing silent videos and videos with audio) or certain codecs. The **FFmpeg export path** explicitly handles mixed sound/no-sound clips and has an integration test for that case.
- No overlapping video tracks, detached music track, clip-to-clip cross-dissolves, video effects, titles with animated templates, or keyframing yet.
- Export is fixed at 1280×720 / 30 fps and letterboxes inputs as needed.
- Title overlay requires FFmpeg built with the `drawtext` filter and suitable system fonts.
- Long timelines may become slow to display as Fyne objects; virtualization will be needed for large projects.
- Linux/Unix mpv preview uses a private Unix-domain socket. Windows/macOS playback support has **not** been tested.

## Tests

```sh
go test ./editor ./media
```

These tests don't need Fyne. `media` integration tests create real sample videos with FFmpeg and verify mixed sound/silence export, title and fade export, audio waveform creation, and mpv EDL serialization. `mpv` itself is not launched by the tests.

**Build verification:** The editor and media packages pass tests in the development environment. The full Fyne app could not be compiled or run there because the Go module dependencies cannot be downloaded and OpenGL headers are unavailable. Its GUI and mpv integration require hands-on validation on a desktop.

To generate sample video assets:

```sh
./scripts/make-demo-clips.sh
```

## Layout

- `main.go`: Fyne app, media library, editing actions, export UI, mpv controls
- `timeline.go`: drag-to-trim/reorder timeline, thumbnail and waveform drawing
- `editor/project.go`: non-destructive clip edit model
- `media/ffmpeg.go`: frame extraction, metadata, final MP4 encoding
- `media/player.go`: mpv EDL generation, separate-window preview and local IPC
- `media/waveform.go`: low-resolution audio peak calculation
- `editor/*_test.go`, `media/*_test.go`: model and FFmpeg tests

## License

GPL-3.0-or-later. See `COPYING`.
