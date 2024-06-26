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

# If the file doesn't exist, create it
if [[ ! -f ${tmp_location} ]]; then
    debug_msg "Creating GOOGLE_APPLICATION_CREDENTIALS file at ${tmp_location}"

    # If the GOOGLE_APPLICATION_CREDENTIALS_CONTENT is not set, use critical_error function
    if [[ -z "${GOOGLE_APPLICATION_CREDENTIALS_CONTENT}" ]]; then
        critical_error "GOOGLE_APPLICATION_CREDENTIALS_CONTENT is not set"
    fi

    echo "${GOOGLE_APPLICATION_CREDENTIALS_CONTENT}" > "${tmp_location}"
fi

# Set the GOOGLE_APPLICATION_CREDENTIALS environment variable
export GOOGLE_APPLICATION_CREDENTIALS=${tmp_location}
debug_msg "GOOGLE_APPLICATION_CREDENTIALS set to ${GOOGLE_APPLICATION_CREDENTIALS}"
