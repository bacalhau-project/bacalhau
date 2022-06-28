#!/bin/bash
set -xeuo pipefail
printf %s\\n {0..99} | xargs -n 1 -P 2 -i \
    hyperfine --export-json=run-$(date +%s%N)-{}.json ./submit.sh
