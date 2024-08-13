#!/bin/bash

echo "$PRIVATE_PEM_B64" | base64 --decode > /tmp/private.pem
echo "$PUBLIC_PEM_B64" | base64 --decode > /tmp/public.pem
export PRIVATE_KEY_PASSPHRASE="$(echo $PRIVATE_KEY_PASSPHRASE_B64 | base64 --decode)"
export BACALHAU_RELEASE_TOKEN=$(buildkite-agent secret get BACALHAU_RELEASE_TOKEN)

# Prevent rebuilding web ui, we should have already attached it
find webui -exec touch -c '{}' +

GOOS=$1 GOARCH=$2 make build-bacalhau-tgz


if [ -z "$BUILDKITE_TAG" ]; then
    cd dist
    echo $BACALHAU_RELEASE_TOKEN | gh auth login --with-token
    gh release upload "1.4.1-dev" bacalhau_*
fi
