#!/usr/bin/env bash

export SHELL=$(command -v bash)
. <(FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau -t;);

go env

# shellcheck disable=SC2312
pre-commit run --show-diff-on-failure --color=always --all-files
