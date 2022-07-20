#!/bin/bash
set -xeuo pipefail

# docker run busybox /bin/true
# /bin/true


BACALHAU_BIN="${1:-../bin/linux_amd64/bacalhau}"
API_PORT="${2:-$(cat /tmp/bacalhau-devstack.port)}"

ID=$(${BACALHAU_BIN} --api-port="${API_PORT}" --api-host=localhost docker run --concurrency=3 busybox -- /bin/true 2>&1)
while true; do
	sleep 0.1
	CURRENT_STATE=$(${BACALHAU_BIN} --api-port="${API_PORT}" --api-host=localhost describe "${ID}" 2>&1 | grep -c 'state:')
	if [[ ${CURRENT_STATE} -ne 3 ]]; then
		echo "JOB ${ID} FAILED"
	else
		echo "JOB ${ID} succeeded"
		exit 0
	fi
done
