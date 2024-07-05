#!/usr/bin/env bash
unset LD_LIBRARY_PATH

# shellcheck disable=SC2155
export SHELL=$(command -v bash)

# shellcheck disable=SC2312,SC1090
. <(FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau -t;);

just build-webui

# Make the build archive dir if it doesn't exist
mkdir -p webui/archive

# Create the build archive
tar -czf webui/archive/webui_build.tar.gz -C webui/build .
