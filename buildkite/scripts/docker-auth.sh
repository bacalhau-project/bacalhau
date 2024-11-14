#!/bin/bash

# Function for docker authentication - can be sourced by other scripts
docker_auth() {
    echo "🔑 Starting Docker Hub authentication..."

    # Function for error handling and logging
    log_error() {
        echo "❌ ERROR: $1" >&2
        exit 1
    }

    log_info() {
        echo "ℹ️ INFO: $1"
    }

    # Try to get secrets from buildkite-agent
    if ! DOCKER_USERNAME=$(buildkite-agent secret get DOCKER_USERNAME 2>/dev/null); then
        log_error "Failed to retrieve DOCKER_USERNAME from buildkite-agent secrets"
    fi

    if ! DOCKER_PASSWORD=$(buildkite-agent secret get DOCKER_PASSWORD 2>/dev/null); then
        log_error "Failed to retrieve DOCKER_PASSWORD from buildkite-agent secrets"
    fi

    # Validate credentials are not empty
    if [ -z "${DOCKER_USERNAME}" ]; then
        log_error "DOCKER_USERNAME is empty or not set"
    fi

    if [ -z "${DOCKER_PASSWORD}" ]; then
        log_error "DOCKER_PASSWORD is empty or not set"
    fi

    # Create docker config directory if it doesn't exist
    mkdir -p ~/.docker || log_error "Failed to create ~/.docker directory"

    # Create or update docker config.json with authentication
    if ! auth_string=$(echo -n "${DOCKER_USERNAME}:${DOCKER_PASSWORD}" | base64); then
        log_error "Failed to generate auth string"
    fi

    cat > ~/.docker/config.json << EOF || log_error "Failed to write Docker config file"
{
    "auths": {
        "https://index.docker.io/v1/": {
            "auth": "${auth_string}"
        }
    }
}
EOF

    # Verify authentication
    if ! echo "${DOCKER_PASSWORD}" | docker login -u "${DOCKER_USERNAME}" --password-stdin > /dev/null 2>&1; then
        log_error "Docker login failed"
    fi

    log_info "Docker authentication successful"
}

# Only run if script is executed directly (not sourced)
if [ "${BASH_SOURCE[0]}" -ef "$0" ]; then
    docker_auth
fi