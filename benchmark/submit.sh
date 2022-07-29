#!/bin/bash
set -xeuo pipefail

# docker run busybox /bin/true
# /bin/true

ID=$(${BACALHAU_BIN} --api-port="${API_PORT}" --api-host=localhost docker run --concurrency=3 busybox -- /bin/true)
COUNTER=0
while true; do
    sleep 0.1
    # trunk-ignore(shellcheck/SC2312)
    if [[ $(${BACALHAU_BIN} --api-port="${API_PORT}" --api-host=localhost describe "${ID}" 2>&1|grep -c 'state:') -ne 3 ]]; then
        echo "JOB ${ID} FAILED"
    else
        echo "JOB ${ID} succeeded"
        exit 0
    fi
    ((COUNTER++))
    if (( COUNTER > 20 )); then
        echo "JOB ${ID} checked ${COUNTER} times. Assuming failure."
        exit 1
    fi
done
