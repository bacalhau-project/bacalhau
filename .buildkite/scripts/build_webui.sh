#!/usr/bin/env bash
unset LD_LIBRARY_PATH

SHELL=$(command -v bash)
export SHELL

# Capture the command's output to a variable
ACTIVATE_COMMAND="FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau -t"

# Source the captured command
. <<<"$ACTIVATE_COMMAND"

just build-webui

# Make the build archive dir if it doesn't exist
mkdir -p webui/archive

# Create the build archive
tar -czf webui/archive/webui_build.tar.gz -C webui/build .
