#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

export LOGFILE=${LOGFILE:="/tmp/bacalhau_lotus_mock_log.txt"}

function hello() {
  echo "Hello, world!"
}

function version() {
  echo "0.0.1"
}

echo "command: $@" >> "$LOGFILE"
eval "$@" >> "$LOGFILE"