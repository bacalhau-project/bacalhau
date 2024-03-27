#!/usr/bin/env bash

# Make sure we are executing in the dev directory
if [ -z "$(echo $PWD | grep '/dev$')" ]; then
    echo "Please execute this script from the dev directory"
    exit 1
fi

# Make sure the python version in tools version is the same as the as the base image for the container
if [ -f ../.tool-versions ]; then
    python_version=$(grep -E "^python " ../.tool-versions | cut -d ' ' -f 2)
    if [ -n "$python_version" ]; then
        python_version=$(echo $python_version | tr -d '\n')
        # Check the base image for the container
        base_image=$(grep -E "^FROM " Dockerfile | cut -d ' ' -f 2)
        if [ -n "$base_image" ]; then
            base_image=$(echo $base_image | tr -d '\n')
            # Check if the python version in the tools version is the same as the base image
            if [[ $base_image != *"$python_version"* ]]; then
                echo "Python version in .tool-versions is not the same as the base image for the container"
                exit 1
            fi
        fi
    fi
fi

# Get tag of latest version
latest_tag=$(git describe --tags $(git rev-list --tags --max-count=1))

# Get golang version from ../.tool-versions and trim whitespace. The entry should look like "golang 1.16.7". Do not print line numbers.
GO_VERSION=$(sed -n 's/golang \([0-9]*\.[0-9]*\.[0-9]*\)/\1/p' ../.tool-versions | tr -d '[:space:]')

GOARCH=$(dpkg --print-architecture)
GOOS=$(uname -s | tr '[:upper:]' '[:lower:]')

GO_ARCHIVE_NAME="go${GO_VERSION}.${GOOS}-${GOARCH}.tar.gz"

# Copy .tool-versions from parent directory to here
cp ../.tool-versions .

# Use custom buildx builder
# First see if the build driver already exists
if [ -z "$(docker buildx ls | grep bacalhau-devcontainer-builder)" ]; then
    docker buildx create --buildkitd-flags '--oci-worker-gc-keepstorage=50000' --use --name bacalhau-devcontainer-builder --append default
fi

# Build docker image with tag of latest tagged version
docker buildx build --platform linux/amd64,linux/arm64 --build-arg "GO_ARCHIVE_NAME=${GO_ARCHIVE_NAME}" --push -t "docker.io/bacalhauproject/bacalhau-devcontainer:${latest_tag}" .