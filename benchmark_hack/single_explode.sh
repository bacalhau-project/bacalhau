#!/bin/bash
set -euo pipefail

export iterationid="$1"

sudo tee ./results/parameters-${RUN_ID}.json > /dev/null <<EOI
{
"TOTAL_JOBS": $TOTAL_JOBS,
"BATCH_SIZE": $BATCH_SIZE,
"CONCURRENCY": $CONCURRENCY
}
EOI

hyperfine \
  --export-json=results/run-${RUN_ID}-$iterationid.json \
  --runs $BATCH_SIZE \
  "timeout 30s ./submit.sh"