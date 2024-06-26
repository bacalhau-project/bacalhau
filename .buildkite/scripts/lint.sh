#!/usr/bin/env bash

# shellcheck disable=SC1090,SC2312
source <(.buildkite/scripts/get_google_application_credentials.sh GOOGLE_APPLICATION_CREDENTIALS_CONTENT)

source <(./.buildkite/scripts/manage_env_secrets.sh get)

SHELL=$(command -v bash) FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau \
                                                        -t -- \
                                                        pre-commit run --show-diff-on-failure --color=always --all-files
