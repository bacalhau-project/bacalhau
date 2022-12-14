#!/bin/bash

# TODO: make this work - in particular, slash the bad compute node in the
# simulator server

set -xeuo pipefail

pkill -f bacalhau || true

bacalhau simulator &

sleep 1

export PREDICTABLE_API_PORT=1
rm -rf /tmp/bacalhau-devstack* ; bacalhau devstack --bad-compute-actors 1 \
    --simulator-url ws://localhost:9075/websocket &

sleep 1

export BACALHAU_API_HOST=localhost
export BACALHAU_API_PORT=20002

while true; do
    bacalhau docker run --verfier deterministic --concurrency 3 ubuntu echo hello
done

# Now, dear human, observe with your eyes that ...

