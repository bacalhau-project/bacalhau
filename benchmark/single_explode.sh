#!/bin/bash
set -euo pipefail

export INTERATION_ID="$1"

# select the port of the next requester node
export API_PORT=$((${API_PORT:-"20000"}  + INTERATION_ID % REQUESTER_NODES))

sudo tee "./results/parameters-${RUN_ID}.json" >/dev/null <<EOI
{
"TOTAL_JOBS": ${TOTAL_JOBS},
"BATCH_SIZE": ${BATCH_SIZE},
"CONCURRENCY": ${CONCURRENCY},
"REQUESTER_NODES": ${REQUESTER_NODES},
}
EOI

hyperfine \
  --ignore-failure \
  --export-json="results/run-${RUN_ID}-${INTERATION_ID}.json" \
  --runs "${BATCH_SIZE}" \
  "timeout 30s bash submit.sh"
