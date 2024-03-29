#!/bin/bash
set -xeuo pipefail
while true; do
  printf %s\\n {0..1000} | xargs -n 1 -P 100 -i \
    "${BACALHAU_BIN}" --api-port="${API_PORT}" --api-host=localhost docker run --concurrency=3 busybox -- /bin/true
done
