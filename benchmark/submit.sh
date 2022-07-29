#!/bin/bash
set -xeuo pipefail

# docker run busybox /bin/true
# /bin/true

ID=$(bacalhau --api-port=$BACALHAU_API_PORT_0 --api-host=localhost docker run --concurrency=3 busybox -- /bin/true)
while true; do
    sleep 0.1
    if [ $(bacalhau --api-port=$BACALHAU_API_PORT_0 --api-host=localhost describe $ID 2>&1|grep "State: Complete"|wc -l) -ne 3 ]; then
        echo "JOB $ID FAILED"
    else
        echo "JOB $ID succeeded"
        exit 0
    fi
done
