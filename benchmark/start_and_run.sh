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

go run .. devstack &

wait_file "/tmp/bacalhau-devstack.pid" 15

./submit.sh
./explode.sh

BACALHAU_PID=$(cat /tmp/bacalhau-devstack.pid)
kill -2 "${BACALHAU_PID}"
