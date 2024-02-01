#!/bin/bash
set -xeuo pipefail

# docker run busybox /bin/true
# /bin/true

ID=$(time ${BACALHAU_BIN} --api-port="${API_PORT}" --api-host=localhost docker run --id-only --concurrency=3 --wait --wait-timeout-secs 20 busybox -- /bin/true)
if [[ $(time ${BACALHAU_BIN} --api-port="${API_PORT}" --api-host=localhost describe "${ID}" |grep "State: Completed"|wc -l) -ne 3 ]]; then
        echo "JOB ${ID} FAILED"
        exit 1
    else
        echo "JOB ${ID} succeeded"
        exit 0
    fi
done
