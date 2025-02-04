#!/bin/bash

set -euo pipefail

set_environment_variables() {
    export GIT_TAG=$(git describe --tags --always)
}

docker_login() {
    export GHCR_PAT=$(buildkite-agent secret get GHCR_PAT)
    echo $GHCR_PAT | docker login ghcr.io -u bacalhau-infra-bot --password-stdin
}

setup_buildx() {
    docker context create buildx-build
    docker buildx create --use buildx-build
}

download_and_extract_artifact() {
    local arch=$1
    local tarball="bacalhau_${GIT_TAG}_linux_${arch}.tar.gz"
    local target_dir="bin/linux/${arch}"

    mkdir -p "$target_dir"
    if ! tar xf "$tarball" -C "$target_dir"; then
        echo "Error: Failed to extract $tarball" >&2
        exit 1
    fi
    echo "Extracted $tarball to $target_dir folder"
}

download_artifacts() {
    echo "--- Downloading build artifacts"
    if ! buildkite-agent artifact download "*.*" . --build "$BUILDKITE_BUILD_ID"; then
        echo "Error: Failed to download artifacts from build pipeline" >&2
        exit 1
    fi

    download_and_extract_artifact "amd64"
    download_and_extract_artifact "arm64"
}

main() {
    if [ -n "${BUILDKITE_TAG:-}" ]; then
        echo "=== Building and pushing images for tag: ${BUILDKITE_TAG}"
        set_environment_variables
        setup_buildx
        download_artifacts
        docker_login
        make push-bacalhau-images
    else
        echo "Skipping image build: BUILDKITE_TAG is not present"
    fi
}

main
