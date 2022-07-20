#!/bin/bash
set -xeuo pipefail

# docker run busybox /bin/true
# /bin/true

API_PORT="$(cat /tmp/bacalhau-devstack.port)"

ID=$(go run .. --api-port="${API_PORT}" --api-host=localhost docker run --concurrency=3 busybox -- /bin/true 2>&1)
while true; do
	sleep 0.1
	CURRENT_STATE=$(go run .. --api-port="${API_PORT}" --api-host=localhost describe "${ID}" 2>&1 | grep -c 'state:')
	if [[ ${CURRENT_STATE} -ne 3 ]]; then
		echo "JOB ${ID} FAILED"
	else
		echo "JOB ${ID} succeeded"
		exit 0
	fi
done
