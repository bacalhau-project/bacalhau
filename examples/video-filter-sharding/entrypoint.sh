#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

inputFolder="$1"
outputFolder="$2"
mkdir -p /tmp/scratch

if [[ -z "$inputFolder" || -z "$outputFolder" ]]; then
  >&2 echo "Usage: $0 <input-folder> <output-folder>"
  exit 1
fi

if [ ! -d "$inputFolder" ]; then
  >&2 echo "Input folder does not exist: $inputFolder"
  exit 1
fi

if [ ! -d "$outputFolder" ]; then
  >&2 echo "Output folder does not exist: $outputFolder"
  exit 1
fi

# given an input video - return a temp path for it
function getScratchVideoPath() {
  local inputVideoFile="$1"
  local outputVideoFilename=$(basename "$inputVideoFile")
  echo "/tmp/scratch/$outputVideoFilename"
}

# given an input video - return the output path for it
function getOutputVideoPath() {
  local inputVideoFile="$1"
  local outputVideoFilename=$(basename "$inputVideoFile")
  echo "$outputFolder/$outputVideoFilename"
}

function convertToAscii() {
  local inputVideoFile="$1"
  local scratchVideoFile=$(getScratchVideoPath "$inputVideoFile")
  ffmpeg -y \
    -i "$inputVideoFile" \
    -vf "datascope=s=1920x1080:mode=color2" \
    -an "$scratchVideoFile"
}

function overlayVideos() {
  local inputVideoFile="$1"
  local scratchVideoFile=$(getScratchVideoPath "$inputVideoFile")
  local outputVideoFile=$(getOutputVideoPath "$inputVideoFile")
  ffmpeg -i "$inputVideoFile" -i "$scratchVideoFile" -filter_complex \
  "[1:0]setdar=dar=1,format=rgba[a]; \
  [0:0]setdar=dar=1,format=rgba[b]; \
  [b][a]blend=all_mode='overlay':all_opacity=0.8" \
  "$outputVideoFile"
}

for filename in $inputFolder/*; do
  convertToAscii "$filename"
  overlayVideos "$filename"
done

ls -la /tmp/scratch
ls -la "$outputFolder"


# #! /bin/sh
# pref="`basename $0 .sh`"
# vleft="zzz_Drifting with Cars.mp4"  # outname of the above script
# vright="Drifting with Cars.mp4"

# #
# fac=${1:-90}
# cx=$((16 * ${fac}))
# cy=$((9 * ${fac}))
# ox=$((1920 - 16 * ${fac}))
# oy=$((1080 - 9 * ${fac}))
# #
# ffmpeg -y -i "${vleft}" -i "${vright}" -filter_complex "
# [0:v]scale=${cx}:${cy},setsar=1,split[0v_1][0v_2];
# [1:v]scale=${cx}:${cy},setsar=1,split[1v_1][1v_2];

# [0v_1]pad=1920:1080:0:0[0v_p];
# [0v_p][1v_1]overlay=x=W-w:y=H-h[v_ov];

# [0v_2]crop=${cx}-${ox}:${cy}-${oy}:${ox}:${oy}[0v_c];
# [1v_2]crop=${cx}-${ox}:${cy}-${oy}:0:0[1v_c];
# [0v_c][1v_c]blend=all_mode=average[v_c];

# [v_ov][v_c]overlay=x=${ox}:y=${oy}[v]" \
#     -map '[v]' -an \
#     "${pref}.mp4"