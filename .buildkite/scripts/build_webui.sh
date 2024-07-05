#!/usr/bin/env bash
unset LD_LIBRARY_PATH

export SHELL=$(command -v bash)
. <(FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau -t;);

# Download vendor and webui/node_modules artifacts
buildkite-agent artifact download "vendor.tar.gz" .
buildkite-agent artifact download "webui_node_modules.tar.gz" .

# Extract the artifacts if they exist
if [ -f vendor.tar.gz ]; then
    tar -xzf vendor.tar.gz
fi

if [ -f webui_node_modules.tar.gz ]; then
    tar -xzf webui_node_modules.tar.gz -C webui/
fi

just build-webui

# Create new artifacts
tar -czf vendor.tar.gz vendor
tar -czf webui_node_modules.tar.gz -C webui/node_modules

# Upload the new artifacts
buildkite-agent artifact upload vendor.tar.gz
buildkite-agent artifact upload webui_node_modules.tar.gz

# Upload the build artifacts
tar -czf webui_build.tar.gz -C webui/build
buildkite-agent artifact upload webui_build.tar.gz
