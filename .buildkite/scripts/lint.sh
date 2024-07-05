#!/usr/bin/env bash

# shellcheck disable=SC2155
export SHELL=$(command -v bash)

# shellcheck disable=SC2312,SC1090
. <(FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau -t;);

go env

# shellcheck disable=SC2312
pre-commit run --show-diff-on-failure --color=always --all-files
