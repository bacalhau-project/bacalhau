#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

export LOTUS_LOGFILE=${LOTUS_LOGFILE:="/tmp/bacalhau_lotus_mock_log.txt"}
export LOTUS_TEST_CONTENT_CID=${LOTUS_TEST_CONTENT_CID:="test-content-cid"}
export LOTUS_TEST_DEAL_CID=${LOTUS_TEST_DEAL_CID:="test-deal-cid"}

function version() {
  echo "0.0.1"
}

function import() {
  echo "Import 3, Root $LOTUS_TEST_CONTENT_CID"
}

function deal() {
  local MINER_ADDRESS="$2"
  cat << EOF
.. executing
Deal (${MINER_ADDRESS}) CID: ${LOTUS_TEST_DEAL_CID}
EOF
}

function client() {
  eval "$@"
}

echo "command: $@" >> "$LOTUS_LOGFILE"
eval "$@" | tee -a "$LOTUS_LOGFILE"