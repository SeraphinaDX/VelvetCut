# VelvetCut 0.8.0 (beta)

VelvetCut is a GPL-3.0-or-later iMovie-inspired editor for Go + Fyne. It uses FFmpeg for non-destructive editing and effect-accurate previews, and mpv to maintain synchronized playback.

## New in 0.8.0 — Video camera motion (Ken Burns effect)

- **Animate the video image:** select a clip and open the **Camera** tab in the docked Clip Inspector. Set a zoom (1×–4×), and a focus position (X=0 left to 1 right; Y=0 top to 1 bottom). Click **Add / update camera keyframe**.
- **Move the playhead** and add another keyframe with different zoom/focus values. VelvetCut smoothly interpolates the framing between them. Outside that interval, it holds the first or last framing.
- **Drag the framing guide:** tick **Drag camera framing in preview**, set zoom above 1×, then drag the outlined rectangle on the paused preview to set a keyframe on release. The guide expresses a normalized focus; the accurate rendered view appears after the edit is applied.
- **Navigate and remove:** previous/next camera keyframes, remove one at the playhead, or clear the camera motion. Camera markers are **cyan**, title markers remain **gold**.
- **Non-destructive:** keyframes are stored within each clip in `.velvetcut` projects; undo/redo, save/load, split, and trim retain the motion. Projects from previous releases still load without changes.
- **Export and preview:** FFmpeg applies the same per-frame `zoompan` motion expressions in paused preview, playback proxies, and MP4 exports. Your video transitions, music and existing title keyframes still work.

### Try it

1. Add a clip with an object or person moving across the shot; select it.
2. Seek to the beginning and open **Camera** in the inspector. Set Zoom to **2×**, Focus horizontal to **0.1**, Focus vertical to **0.5**, and click **Add / update camera keyframe**.
3. Seek two seconds forward, set Focus horizontal to **0.9**, and click **Add / update camera keyframe** again.
4. Scrub through the clip, press **Play**, and export a short MP4 to confirm the camera pan matches.
5. Try a different zoom level at the second keyframe, or tick the drag checkbox and reposition the frame directly in the paused preview.

### Known limitations

The Camera tab animates pan and zoom **within one video track**. Opacity animation, stacked picture-in-picture video layers, and a full multi-layer compositor are not included in this release. The framing guide is a control overlay, not a full unzoomed source monitor. On large projects, effects-accurate preview still requires an FFmpeg proxy render. Full Fyne GUI interaction has not been verified in this build environment.

## New in 0.7.0 — animated titles and direct positioning

- **Drag a selected title's purple outline directly over the video preview.** Release to store its position as a keyframe at the playhead. The video itself remains unmodified. If the title is not visible, first type a title in the Inspector's Title tab and click **Apply**.
- **Motion tab in the Clip Inspector:** edit title scale (0.25×–4×), opacity (0%–100%), and add/update a keyframe at the current playhead. Jump to previous/next keyframes, remove the one at the playhead, or clear all motion.
- **Linear interpolation:** title position, scale, and opacity change smoothly between keyframes, holding their first/last values outside the animated interval.
- **Timeline markers:** small gold keyframe marks appear within each video clip, at clip-local times.
- **Non-destructive & undoable:** moving a title creates one undoable edit on mouse release. Keyframes survive saving/reopening `.velvetcut` projects, and are adjusted when the owning clip is trimmed or split. Old projects load unchanged.
- **Accurate render path:** the same FFmpeg `drawtext` motion expressions are used in exported MP4s, 15fps composited playback proxies and paused preview stills. When motion keyframes exist, they override legacy title entrance animations (fade/slide/zoom); the legacy settings still work when no keyframes are present.

### Try it

1. Import a video and insert it on the timeline.
2. Select the clip, enter a title in the Title inspector, and click **Apply to selected clip**.
3. Seek near the beginning, drag the purple title outline toward the left and release.
4. Seek later, drag it toward the right, then open **Motion** and change the scale/opacity before clicking **Add / update keyframe**.
5. Press Play (the editor renders a cached composited preview), then Export MP4. Keyframes remain within the selected clip.

## Other capabilities

- Split, trim, drag-to-reorder clips; transitions, cross-dissolves and fade through black.
- Multiple audio lanes with volume, fades, mute, independent audio trims, soundtrack detaching.
- Colour correction, black-and-white, sepia, vignette, rotation, cropping and title styling.
- Save/open `.velvetcut` JSON project files; H.264/AAC export; undo/redo and keyboard shortcuts.

## Running on CachyOS / Arch

```sh
sudo pacman -S --needed go ffmpeg mpv base-devel mesa libglvnd libx11 libxcursor libxrandr libxinerama libxi
go mod tidy
go run .
```

Run package tests with:

```sh
go test -race ./editor ./media
```

## Limitations

- Title motion and camera motion are **clip-local**; arbitrary stacked composited overlay tracks are not yet implemented.
- Preview dragging shows an overlay guide during the drag; the image is rendered with its new title position on release. Playback uses a cached FFmpeg proxy rather than a real-time GPU compositor.
- Title keyframes take effect only while the title is visible (controlled by the Title duration setting, up to 30 seconds in the project format).
- The GUI and Fyne interaction require testing on a Linux desktop. Windows/macOS have not been validated.
- Imported video/audio remain referenced by path and are not copied into a project.

Licensed GPL-3.0-or-later; see `COPYING`.
