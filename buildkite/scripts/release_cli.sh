#!/bin/bash

set -euo pipefail

set_environment_variables() {
    export BACALHAU_RELEASE_TOKEN=$(buildkite-agent secret get BACALHAU_RELEASE_TOKEN)
    echo "Fetched Released Token"
}

download_artifact() {
    buildkite-agent artifact download "*.*" .  --build $BUILDKITE_BUILD_ID
    echo "Downloaded artifacts from build pipeline"
}


upload_artifact_to_github() {
    echo "$BACALHAU_RELEASE_TOKEN" | gh auth login --with-token

    if [ -z "${BUILDKITE_TAG:-}" ]; then
        echo "Tag is Missing"
        exit 1
    fi

    gh release upload "$BUILDKITE_TAG" bacalhau_"${BUILDKITE_TAG}"*
}

main() {
    set_environment_variables
    download_artifact
    upload_artifact_to_github
}

main
