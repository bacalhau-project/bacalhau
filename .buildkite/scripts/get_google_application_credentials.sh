#!/usr/bin/env bash

# Function to display debug messages
debug_mode=0
function debug_msg {
    if [[ $debug_mode -eq 1 ]]; then
        echo "$1"
    fi
}

# Function to display critical error messages and exit
function critical_error {
    echo "Critical error: $1"
    exit 1
}

# Check for --debug flag
if [[ "$1" == "--debug" ]]; then
    debug_mode=1
    debug_msg "Debug mode is ON."
fi

tmp_location="${GOOGLE_APPLICATION_CREDENTIALS:-'/tmp/bacalhau-buildkite-sa@bacalhau-infra.iam.gserviceaccount.com-credentials.json'}"

# Get the first argument of the script - show an error if not set
if [[ -z "$1" ]]; then
    critical_error "No secret name for BuildKite agent set"
fi

secret_name=$1

# If the file doesn't exist, create it
if [[ ! -f ${tmp_location} ]]; then
    debug_msg "Creating GOOGLE_APPLICATION_CREDENTIALS file at ${tmp_location}"

    secret_contents=$(buildkite-agent buildkite-agent secret get "${secret_name}")

    # If the secret_contents is not set, use critical_error function
    if [[ -z "${secret_contents}" ]]; then
        critical_error "$secret_name is empty or not set"
    fi

    echo "${secret_contents}" > "${tmp_location}"
fi

# Set the GOOGLE_APPLICATION_CREDENTIALS environment variable
export GOOGLE_APPLICATION_CREDENTIALS=${tmp_location}
debug_msg "GOOGLE_APPLICATION_CREDENTIALS set to ${GOOGLE_APPLICATION_CREDENTIALS}"
