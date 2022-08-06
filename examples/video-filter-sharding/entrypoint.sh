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
  local prefix="$1"
  local inputVideoFile="$2"
  local outputVideoFilename=$(basename "$inputVideoFile")
  echo "/tmp/scratch/${prefix}-${outputVideoFilename}"
}

# given an input video - return the output path for it
function getOutputVideoPath() {
  local inputVideoFile="$1"
  local outputVideoFilename=$(basename "$inputVideoFile")
  echo "$outputFolder/$outputVideoFilename"
}

function resizeVideo() {
  local inputVideoFile="$1"
  local outputVideoFile="$2"
  local scale="$3"
  ffmpeg -i "$inputVideoFile" -vf scale=$scale,setsar=1:1 "$outputVideoFile"
}

function convertToAscii() {
  local inputVideoFile="$1"
  local outputVideoFile="$2"
  local scale="$3"
  ffmpeg -y \
    -i "$inputVideoFile" \
    -vf "datascope=s=$scale:mode=color2" \
    -an "$outputVideoFile"
}

function overlayVideos() {
  local inputVideoFile1="$1"
  local inputVideoFile2="$2"
  local outputVideoFile="$3"
  ffmpeg -i "$inputVideoFile1" -i "$inputVideoFile2" -filter_complex \
  "[1:0]setdar=dar=1,format=rgba[a]; \
  [0:0]setdar=dar=1,format=rgba[b]; \
  [b][a]blend=all_mode='overlay':all_opacity=0.8" \
  "$outputVideoFile"
}

for filename in $inputFolder/*; do
  asciiFilePath=$(getScratchVideoPath "ascii" "$filename")
  outputPath=$(getOutputVideoPath "$filename")
  # resizeVideo "$filename" "$outputPath" "352:240"
  convertToAscii "$filename" "$asciiFilePath" "352x240"
  overlayVideos "$filename" "$asciiFilePath" "$outputPath"
done
