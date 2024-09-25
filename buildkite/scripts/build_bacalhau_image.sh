#!/bin/bash

set -e


download_artifact() {
    if ! buildkite-agent artifact download "*.*" . --build $BUILDKITE_BUILD_ID; then
        echo "Error: Failed to download artifacts from build pipeline" >&2
        exit 1
    fi
    echo "Downloaded artifacts from build pipeline"

    mkdir -p bacalhau_linux_amd64
    if ! tar xf bacalhau_linux_amd64.tar.gz -C bacalhau_linux_amd64; then
        echo "Error: Failed to extract bacalhau_linux_amd64.tar.gz" >&2
        exit 1
    fi
    echo "Extracted bacalhau_linux_amd64.tar.gz to bacalhau_linux_amd64 folder"

    # Add extraction for ARM version
    mkdir -p bacalhau_linux_arm64
    if ! tar xf bacalhau_linux_arm64.tar.gz -C bacalhau_linux_arm64; then
        echo "Error: Failed to extract bacalhau_linux_arm64.tar.gz" >&2
        exit 1
    fi
    echo "Extracted bacalhau_linux_arm64.tar.gz to bacalhau_linux_arm64 folder"
}



main() {
    if [ -z "$BUILDKITE_TAG" ]; then
        download_artifact
    else
        echo "Skipping artifact download: BUILDKITE_TAG is present"
    fi
}