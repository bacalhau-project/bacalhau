#!/usr/bin/env bash

# Make sure we are executing in the dev directory
if [ -z "$(echo $PWD | grep '/dev$')" ]; then
    echo "Please execute this script from the dev directory"
    exit 1
fi

# Get tag of latest version
latest_tag=$(git describe --tags $(git rev-list --tags --max-count=1))

# Get golang version from ../.tool-versions and trim whitespace. The entry should look like "golang 1.16.7". Do not print line numbers.
GO_VERSION=$(sed -n 's/golang \([0-9]*\.[0-9]*\.[0-9]*\)/\1/p' ../.tool-versions | tr -d '[:space:]')

# Copy .tool-versions from parent directory to here
cp ../.tool-versions .
cp ../pyproject.toml .
cp ../poetry.lock .

# Use custom buildx builder
# First see if the build driver already exists
if [ -z "$(docker buildx ls | grep bacalhau-devcontainer-builder)" ]; then
    docker buildx create --buildkitd-flags '--oci-worker-gc=false' --use --name bacalhau-devcontainer-builder --append default 
fi

# Build docker image with tag of latest tagged version
docker buildx build --platform linux/amd64,linux/arm64 --build-arg "GO_VERSION=${GO_VERSION}" --push -t "docker.io/bacalhauproject/bacalhau-devcontainer:${latest_tag}" .