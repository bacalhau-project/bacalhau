#!/bin/bash

# Function to display usage and exit
usage() {
    echo "Usage: $0 -c \"command with args\" (Required)"
    echo "Example: $0 -c \"bacalhau server --config=...\""
    exit 1
}

# Check if no arguments were provided
if [[ $# -eq 0 ]]; then
    echo "Error: Command argument is required"
    usage
fi

# Default empty custom command
COMPUTE_NODE_STARTUP_COMMAND=""

# Parse command line arguments
while getopts ":c:" opt; do
    # shellcheck disable=SC2249
    case ${opt} in
        c)
            # Store everything after -c as the custom command
            COMPUTE_NODE_STARTUP_COMMAND="${OPTARG}"
            ;;
        \?) # Invalid option
            echo "Error: Invalid option -${OPTARG}"
            usage
            ;;
        :) # Option missing required argument
            echo "Error: Option -${OPTARG} requires an argument"
            usage
            ;;
    esac
done

# Check if command was provided
if [[ -z "${COMPUTE_NODE_STARTUP_COMMAND}" ]]; then
    echo "Error: Command argument (-c) is required"
    usage
fi

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

        echo "Executing custom command: ${CUSTOM_COMMAND}"
        eval "${COMPUTE_NODE_STARTUP_COMMAND}" &

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
