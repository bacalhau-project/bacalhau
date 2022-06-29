#!/bin/bash
set -xeuo pipefail
printf %s\\n {0..9999} | xargs -n 1 -P 10 -i \
    hyperfine --export-json=run-$(date +%s%N)-{}.json ./submit.sh
