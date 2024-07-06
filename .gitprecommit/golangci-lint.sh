#!/usr/bin/env bash

all_files=$(golangci-lint run --allow-parallel-runners \
    --timeout 10m \
    --concurrency=10 \
    --config=.golangci.yml \
    --verbose \
    --print-issued-lines=false \
    --print-linter-name=false)

echo "${all_files}"
