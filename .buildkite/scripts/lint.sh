#!/usr/bin/env bash

go env

# If no BUILDKITE_JOB_ID then it's not running in CI
if [ -z "$BUILDKITE_JOB_ID" ]; then
    # shellcheck disable=SC1090,SC2312
    source <(.buildkite/scripts/get_google_application_credentials.sh GOOGLE_APPLICATION_CREDENTIALS_CONTENT)

    # shellcheck disable=SC1090,SC2312
    source <(./.buildkite/scripts/manage_env_secrets.sh get)
fi

# Need to build webui to check lint
just build-webui

# shellcheck disable=SC2312
SHELL=$(command -v bash) FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau \
                                                        -t -- \
                                                        pre-commit run --show-diff-on-failure --color=always --all-files
