#!/usr/bin/env bash

SHELL=$(command -v bash)
export SHELL

# Capture the output of the command
output=$(FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau -t)

# Check if the command was successful
if [ $? -eq 0 ]; then
    # Source the captured output
    . <<< "${output}"
else
    echo "Failed to activate flox environment."
    exit 1
fi

go env

# shellcheck disable=SC2312
pre-commit run --show-diff-on-failure --color=always --all-files
go env
