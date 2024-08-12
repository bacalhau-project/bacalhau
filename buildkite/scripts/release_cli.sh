
#!/bin/bash
set -e

set_environment_variables() {
    export BACALHAU_RELEASE_TOKEN=$(buildkite-agent secret get BACALHAU_RELEASE_TOKEN)
}

download_artifact() {
    bacalhau-agent artifact download "bacalhau_*"
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
