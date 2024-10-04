#!/bin/bash

# Total wait time in seconds
TOTAL_WAIT_TIME_FOR_DOCKERD=30
# Time interval between retries in seconds
RETRY_INTERVAL=3
# Calculating the maximum number of attempts
MAX_ATTEMPTS=$((TOTAL_WAIT_TIME_FOR_DOCKERD / RETRY_INTERVAL))
attempt=1

# Start the first docker daemon
dockerd-entrypoint.sh &

while [[ ${attempt} -le ${MAX_ATTEMPTS} ]]; do
    echo "Checking if dockerd is available (attempt ${attempt} of ${MAX_ATTEMPTS})..."

    # Try to communicate with Docker daemon
    if docker info >/dev/null 2>&1; then
        echo "dockerd is available! Now Starting Bacalhau as a compute node"
        bacalhau config set "node.network.authsecret" "${NETWORK_AUTH_TOKEN}"
        bacalhau serve --node-type=compute --orchestrators="nats://${REQUESTER_NODE_LINK}:4222"
        # Wait for any process to exit
        wait -n

        # Exit with status of process that exited first
        exit $?
    fi

    # Wait before retrying
    echo "dockerd is not available yet. Retrying in ${RETRY_INTERVAL} seconds..."
    sleep "${RETRY_INTERVAL}"

    # Increment attempt counter
    attempt=$((attempt + 1))
done

echo "dockerd did not become available within ${TOTAL_WAIT_TIME_FOR_DOCKERD} seconds."
exit 1
