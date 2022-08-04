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
