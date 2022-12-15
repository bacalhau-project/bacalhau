#!/bin/bash

# TODO: make this work - in particular, slash the bad compute node in the
# simulator server

set -xeuo pipefail

pkill -f bacalhau || true

sleep 1

export PREDICTABLE_API_PORT=1
rm -rf /tmp/bacalhau-devstack* ; bacalhau devstack --bad-compute-actors 1 \
    --simulator-mode &

sleep 3

export BACALHAU_API_HOST=localhost
export BACALHAU_API_PORT=20000

while true; do
    bacalhau docker run --verifier deterministic --concurrency 3 ubuntu echo hello
done

# Now, dear human, observe with your eyes that ...

