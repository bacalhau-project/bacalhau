#!/usr/bin/env bash
set -euo pipefail

export TOTAL_JOBS=${TOTAL_JOBS:-"50"}
export BATCH_SIZE=${BATCH_SIZE:-"10"}
export CONCURRENCY=${CONCURRENCY:-"2"}
export REQUESTER_NODES=${REQUESTER_NODES:-"2"}
export XARGS_LOOPS=$(( TOTAL_JOBS / BATCH_SIZE ))
# trunk-ignore(shellcheck/SC2155)
export RUN_ID=$(date +%s%N)

mkdir -p results

(for ((i=0; i<XARGS_LOOPS; i++)); do echo "${i}"; done) | xargs -P "${CONCURRENCY}" -I{} \
bash single_explode.sh {}

exit 0
