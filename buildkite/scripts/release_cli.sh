
#!/bin/bash
set -e

set_environment_variables() {
    export BACALHAU_RELEASE_TOKEN=$(buildkite-agent secret get BACALHAU_RELEASE_TOKEN)
    echo "Fetched Released Token"
}

download_artifact() {
    buildkite-agent artifact download "*.*" .  --build "$(buildkite-agent meta-data get "triggered_build_id")"
    echo "Downloaded artifacts from build pipeline"
}


upload_artifact_to_github() {
    echo "$BACALHAU_RELEASE_TOKEN" | gh auth login --with-token

    if [ -z "$BUILDKITE_TAG" ]; then
        echo "Tag is Missing"
        exit 1
    fi

    gh release upload $TAG bacalhau_$TAG_*
}

main() {
    set_environment_variables
    download_artifact
    upload_artifact_to_github
}
