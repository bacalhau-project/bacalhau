#!/bin/bash
set -xeuo pipefail
mkdir -p results
printf %s\\n {0..9999} | xargs -n 1 -P 10 -i \
    hyperfine --ignore-failure --export-json=results/run-$(date +%s%N)-{}.json "timeout 30s ./submit.sh"
