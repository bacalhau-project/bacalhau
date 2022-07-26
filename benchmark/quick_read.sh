#!/bin/bash
set -xeuo pipefail

while true; do
  printf %s\\n {0..1000} | xargs -n 1 -P 100 -i \
    bacalhau --api-port=$BACALHAU_API_PORT_0 --api-host=localhost list --wide
  sleep 1
done
