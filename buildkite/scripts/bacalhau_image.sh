#!/bin/bash

set -euo pipefail

set_environment_variables() {
    export GIT_TAG=$(git describe --tags --always)
}

docker_login() {
    export GHCR_PAT=$(buildkite-agent secret get GHCR_PAT)
    echo $GHCR_PAT | docker login ghcr.io -u bacalhau-infra-bot --password-stdin
}

docker_context_create() {
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
    if ! buildkite-agent artifact download "*.*" . --build "$BUILDKITE_BUILD_ID"; then
        echo "Error: Failed to download artifacts from build pipeline" >&2
        exit 1
    fi
    echo "Downloaded artifacts from build pipeline"

    download_and_extract_artifact "amd64"
    download_and_extract_artifact "arm64"
}

main() {
    if [ -z "${BUILDKITE_TAG:-}" ]; then
        set_environment_variables
        docker_context_create
        download_artifacts
        make build-bacalhau-image
        docker_login
        make push-bacalhau-image
    else
        echo "Skipping artifact download: BUILDKITE_TAG is present"
    fi
}

main
