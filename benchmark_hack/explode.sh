#!/bin/bash
set -xeuo pipefail

export TOTAL_JOBS=${TOTAL_JOBS:-"10000"}
export BATCH_SIZE=${BATCH_SIZE:-"10"}
export CONCURRENCY=${CONCURRENCY:-"10"}

mkdir -p results
printf %s\\n {0..$(( $TOTAL_JOBS / $BATCH_SIZE ))} | xargs -n 1 -P $CONCURRENCY -i \
    hyperfine \
      --export-json=results/run-$(date +%s%N)-{}.json \
      --runs $BATCH_SIZE \
      ./submit.sh
