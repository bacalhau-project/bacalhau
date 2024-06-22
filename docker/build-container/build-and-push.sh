#!/usr/bin/env bash

set -ex

CHECKOUT_BRANCH=${CHECKOUT_BRANCH:-main}
BUILD_DIR=${BUILD_DIR:-.}
BUILD_NAME=${BUILD_NAME:-build-container}

# If BUILD_DIR is not the current directory, change to it
if [ "$BUILD_DIR" != "." ]; then
    cd $BUILD_DIR
fi

# Read the VERSION
VERSION=$(cat VERSION)

# Increment the version
MAJOR=$(echo $VERSION | cut -d. -f1)
MINOR=$(echo $VERSION | cut -d. -f2)
PATCH=$(echo $VERSION | cut -d. -f3)
PATCH=$((PATCH + 1))
NEW_VERSION="$MAJOR.$MINOR.$PATCH"

# Write the new version
echo $NEW_VERSION > VERSION

# Build the new version container
docker buildx build --build-arg=BRANCH="${CHECKOUT_BRANCH}" --platform linux/amd64 --push -t docker.io/bacalhauproject/$BUILD_NAME:$NEW_VERSION .
docker pull --platform linux/amd64 docker.io/bacalhauproject/$BUILD_NAME:$NEW_VERSION

docker tag docker.io/bacalhauproject/$BUILD_NAME:$NEW_VERSION docker.io/bacalhauproject/$BUILD_NAME:latest
docker push docker.io/bacalhauproject/$BUILD_NAME:latest
