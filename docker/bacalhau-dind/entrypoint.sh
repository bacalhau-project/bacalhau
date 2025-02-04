#!/bin/sh

# Check for privileged mode by testing iptables access
if ! iptables -L >/dev/null 2>&1; then
    echo "ERROR: This container must be run with --privileged flag"
    echo "Example: docker run --privileged <image> serve"
    exit 1
fi

# Start the Docker daemon
dockerd-entrypoint.sh dockerd &

# Wait for Docker daemon with timeout
timeout 30s sh -c 'until docker info > /dev/null 2>&1; do echo "Waiting for Docker daemon..."; sleep 1; done'

if [ $? -ne 0 ]; then
    echo "Timed out waiting for Docker daemon"
    exit 1
fi

echo "Docker daemon is ready"

# Get the bacalhau binary path (first argument)
BACALHAU_BIN=$1
shift

# Execute bacalhau with the remaining arguments
exec "$BACALHAU_BIN" "$@"