#!/bin/sh

set -e

# Check for privileged mode by testing iptables access
if ! iptables -L >/dev/null 2>&1; then
    echo "ERROR: This container must be run with --privileged flag"
    echo "Example: docker run --privileged <image> serve"
    exit 1
fi

# Add initial random delay (0-2000 milliseconds) to prevent thundering herd
HOSTNAME=$(hostname)
INITIAL_DELAY_MS=$(awk -v seed="$(echo $HOSTNAME | cksum | cut -d' ' -f1)" 'BEGIN{srand(seed);print int(rand()*2000)}')
echo "Adding initial startup delay of ${INITIAL_DELAY_MS} milliseconds..."
sleep "$(awk "BEGIN{print ${INITIAL_DELAY_MS}/1000.0}")"

MAX_RETRIES=5
ATTEMPT=1
TIMEOUT=45

start_docker_daemon() {
    # Kill any existing Docker daemon process
    killall containerd dockerd >/dev/null 2>&1 || true
    sleep 0.5  # Brief pause for cleanup
    
    # Clean up any existing socket
    rm -f /var/run/docker.pid /var/run/docker.sock /run/containerd/containerd.sock
    
    # Start the Docker daemon in the background
    echo "Starting Docker daemon (Attempt $ATTEMPT/$MAX_RETRIES)..."
    dockerd-entrypoint.sh dockerd > /var/log/dockerd.log 2>&1 &
    DOCKERD_PID=$!
    
    # Wait for initialization
    start_time=$(date +%s)
    while [ $(( $(date +%s) - start_time )) -lt $TIMEOUT ]; do
        if docker info >/dev/null 2>&1; then
            echo "Docker daemon is ready"
            return 0
        fi

        if ! kill -0 $DOCKERD_PID 2>/dev/null; then
            echo "Docker daemon process died during attempt $ATTEMPT"
            echo "Docker daemon logs:"
            cat /var/log/dockerd.log
            return 1
        fi

        elapsed_time=$(( $(date +%s) - start_time ))
        echo "Waiting for Docker daemon... (${elapsed_time}/${TIMEOUT} seconds)"
        sleep 0.5
    done
    
    echo "Docker daemon failed to start within timeout"
    return 1
}

# Try to start Docker daemon with retries
while [ $ATTEMPT -le $MAX_RETRIES ]; do
    if start_docker_daemon; then
        break
    fi
    
    ATTEMPT=$((ATTEMPT + 1))
    if [ $ATTEMPT -le $MAX_RETRIES ]; then
        echo "Retrying Docker daemon startup after 1 second..."
        sleep 1
    fi
done

if [ $ATTEMPT -gt $MAX_RETRIES ]; then
    echo "ERROR: Failed to start Docker daemon after $MAX_RETRIES attempts"
    echo "Final Docker daemon logs:"
    cat /var/log/dockerd.log
    exit 1
fi

# Get the bacalhau binary path (first argument)
BACALHAU_BIN=$1
shift

# Execute bacalhau with the remaining arguments
exec "$BACALHAU_BIN" "$@"