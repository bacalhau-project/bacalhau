#!/bin/bash
set -xeuo pipefail

pkill -f bacalhau || true

bacalhau simulator &

sleep 1

export PREDICTABLE_API_PORT=1
rm -rf /tmp/bacalhau-devstack* ; bacalhau devstack --simulator-url ws://localhost:9075/websocket &

sleep 1

export BACALHAU_API_HOST=localhost
export BACALHAU_API_PORT=20002

while true; do bacalhau docker run --concurrency 3 ubuntu echo hello; done

# Now, dear human, observe with your eyes that the jobs stop working after the
# requestor node balance is drained. This demonstrates that the "stop working
# when you've run out of money" behavior is working.

