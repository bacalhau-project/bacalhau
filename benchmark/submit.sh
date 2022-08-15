#!/bin/bash
set -xeuo pipefail

# docker run busybox /bin/true
# /bin/true

ID=$(${BACALHAU_BIN} --api-port="${API_PORT}" --api-host=localhost docker run --concurrency=3 busybox -- /bin/true)
COUNTER=1
while true; do
    sleep 0.1
    # trunk-ignore(shellcheck/SC2312)
    # TODO: get the shard state to not be a number (which is brittle to test against)
    if [[ $(${BACALHAU_BIN} --api-port="${API_PORT}" --api-host=localhost describe "${ID}" 2>&1|grep "State: Complete"|wc -l) -ne 3 ]]; then
        echo "JOB ${ID} FAILED"
        exit 1
    else
        echo "JOB ${ID} succeeded"
        exit 0
    fi
done
