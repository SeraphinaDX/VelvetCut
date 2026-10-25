#!/usr/bin/env bash
set -euo pipefail
mkdir -p demo-media
ffmpeg -hide_banner -loglevel error -y -f lavfi -i "testsrc2=size=640x360:rate=30:duration=5" \
  -f lavfi -i "sine=frequency=440:sample_rate=48000:duration=5" \
  -c:v libx264 -pix_fmt yuv420p -c:a aac -shortest demo-media/demo-one.mp4
ffmpeg -hide_banner -loglevel error -y -f lavfi -i "testsrc2=size=640x360:rate=30:duration=5" \
  -vf hflip -c:v libx264 -pix_fmt yuv420p demo-media/demo-two.mp4
printf 'Created demo-media/demo-one.mp4 and demo-media/demo-two.mp4\n'
