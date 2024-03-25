#!/usr/bin/env bash

# Get tag of latest version
latest_tag=$(git describe --tags $(git rev-list --tags --max-count=1))

# Get golang version from ../.tool-versions and trim whitespace. The entry should look like "golang 1.16.7". Do not print line numbers.
GO_VERSION=$(sed -n 's/golang \([0-9]*\.[0-9]*\.[0-9]*\)/\1/p')

GOARCH=$(dpkg --print-architecture)
GOOS=$(uname -s | tr '[:upper:]' '[:lower:]')

GO_ARCHIVE_NAME="go${GO_VERSION}.${GOOS}-${GOARCH}.tar.gz"

# Use custom buildx builder
docker buildx create --buildkitd-flags '--oci-worker-gc-keepstorage=50000' --use --name bacalhau-devcontainer-builder --append default

# Build docker image with tag of latest tagged version
docker buildx build --platform linux/amd64,linux/arm64 --build-arg "GO_ARCHIVE_NAME=${GO_ARCHIVE_NAME}" --push -t "docker.io/bacalhauproject/bacalhau-devcontainer:${latest_tag}" .