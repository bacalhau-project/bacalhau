#!/usr/bin/env bash
unset LD_LIBRARY_PATH

export SHELL=$(command -v bash)
. <(FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau -t;);

just build-webui

# Sleep for 10 minutes
sleep 600

# Create new artifacts
tar -czf vendor.tar.gz vendor
tar -czf webui_node_modules.tar.gz -C webui/node_modules

# Upload the new artifacts
buildkite-agent artifact upload vendor.tar.gz
buildkite-agent artifact upload webui_node_modules.tar.gz

# Upload the build artifacts
tar -czf webui_build.tar.gz -C webui/build
buildkite-agent artifact upload webui_build.tar.gz
