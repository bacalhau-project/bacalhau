#!/usr/bin/env bash

set -e

flox push

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
docker build --push -t docker.io/bacalhauproject/build-container:$NEW_VERSION .
