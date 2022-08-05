#!/bin/bash
set -xeuo pipefail

# Block until the given file appears or the given timeout is reached.
# Exit status is 0 iff the file exists.
wait_file() {
	local file="$1"
	shift
	local wait_seconds="${1:-10}"
	shift # 10 seconds as default timeout
	test "${wait_seconds}" -lt 1 && echo 'At least 1 second is required' && return 1

	until test $((wait_seconds--)) -eq 0 -o -e "${file}"; do sleep 1; done

	test "${wait_seconds}" -ge 0 # equivalent: let ++wait_seconds
}

function cleanup {
	echo "Done. Exiting normally."
	if [ -f "/tmp/bacalhau-devstack.pid" ]; then
		BACALHAU_PID=$(cat /tmp/bacalhau-devstack.pid)
		kill -2 "${BACALHAU_PID}"
	fi
	rm -f /tmp/bacalhau-devstack.p*
	exit 0 
}

trap cleanup EXIT

cd ..
make build

cd benchmark
export BACALHAU_BIN=${BACALHAU_BIN:-"../bin/linux_amd64/bacalhau"}
export PREDICTABLE_API_PORT=1

${BACALHAU_BIN} devstack --nodes 250 &

wait_file "/tmp/bacalhau-devstack.pid" 1500

# trunk-ignore(shellcheck/SC2155)
export API_PORT="$(cat /tmp/bacalhau-devstack.port)"

# ./submit.sh "${BACALHAU_BIN}" "${API_PORT}"
./explode.sh "${BACALHAU_BIN}" "${API_PORT}"

while : ; do
	sleep 2
	CURRENT_STATE=$(${BACALHAU_BIN} --api-port="${API_PORT}" --api-host=localhost list -n 10000 2>&1 | grep -c -E 'Running|Waiting')
	(( CURRENT_STATE > 0 )) || break
done 

echo "Finished. Cleaning up..."
