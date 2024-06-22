#!/usr/bin/env bash

set -e

CHECKOUT_BRANCH=${CHECKOUT_BRANCH:-main}

# Read the VERSION
VERSION=$(cat VERSION)

cp ../../go.mod .
cp ../../go.sum .
cp ../../requirements.txt .

# Increment the version
MAJOR=$(echo $VERSION | cut -d. -f1)
MINOR=$(echo $VERSION | cut -d. -f2)
PATCH=$(echo $VERSION | cut -d. -f3)
PATCH=$((PATCH + 1))
NEW_VERSION="$MAJOR.$MINOR.$PATCH"

# Write the new version
echo $NEW_VERSION > VERSION

# Build the new version container
docker buildx build --build-arg=BRANCH="${CHECKOUT_BRANCH}" --platform linux/amd64 --push -t docker.io/bacalhauproject/build-container:$NEW_VERSION .

docker pull docker.io/bacalhauproject/build-container:$NEW_VERSION
