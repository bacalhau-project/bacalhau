#!/bin/bash

echo "$PRIVATE_PEM_B64" | base64 --decode > /tmp/private.pem
echo "$PUBLIC_PEM_B64" | base64 --decode > /tmp/public.pem
export PRIVATE_KEY_PASSPHRASE="$(echo $PRIVATE_KEY_PASSPHRASE_B64 | base64 --decode)"
# Prevent rebuilding web ui, we should have already attached it
find webui -exec touch -c '{}' +

GOOS=$1 GOARCH=$2 make build-bacalhau-tgz


if [ -z "$BUILDKITE_TAG" ]; then
    buildkite-agent artifact upload "dist/bacalhau_*"
    echo buildkite-agent meta-data set "triggered_build_id" $BUILDKITE_BUILD_ID --build $BUILDKITE_TRIGGERED_FROM_BUILD_ID
    buildkite-agent meta-data set "triggered_build_id" $BUILDKITE_BUILD_ID --build $BUILDKITE_TRIGGERED_FROM_BUILD_ID
fi
