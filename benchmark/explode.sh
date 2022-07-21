#!/bin/bash
set -euo pipefail

export TOTAL_JOBS=${TOTAL_JOBS:-"50"}
export BATCH_SIZE=${BATCH_SIZE:-"10"}
export CONCURRENCY=${CONCURRENCY:-"2"}
export XARGS_LOOPS=$(( $TOTAL_JOBS / $BATCH_SIZE ))
export RUN_ID=$(date +%s%N)

mkdir -p results

(for ((i=0; i<$XARGS_LOOPS; i++)); do echo $i; done) | xargs -P $CONCURRENCY -I{} \
  bash single_explode.sh {} $1 $2

exit 0
