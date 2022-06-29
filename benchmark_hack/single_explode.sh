#!/bin/bash
set -xeuo pipefail

export iterationid="$1"
export runid=$(date +%s%N)
sudo tee ./results/parameters-$runid.json > /dev/null <<"EOI"
{
"TOTAL_JOBS": $TOTAL_JOBS,
"BATCH_SIZE": $BATCH_SIZE,
"CONCURRENCY": $CONCURRENCY
}
EOI

hyperfine \
  --export-json=results/run-$runid-$iterationid.json \
  --runs $BATCH_SIZE \
  ./submit.sh