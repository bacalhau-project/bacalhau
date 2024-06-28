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
    shift
fi

# Get the first argument of the script - show an error if not set
if [[ -z "$1" ]]; then
    critical_error "No secret name for BuildKite agent set"
fi

secret_name=$1

credentials_location="/etc/buildkite-agent/bacalhau-buildkite-sa@bacalhau-infra.iam.gserviceaccount.com-credentials.json"

# Function to install the latest gcloud CLI
install_gcloud() {
    echo "Installing the latest gcloud CLI..."
    echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
    curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg
    sudo apt-get update && sudo apt-get install -y google-cloud-sdk
}

# Function to check if the user is authenticated with Google Cloud
check_gcloud_authentication() {
    if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q .; then
        critical_error "User is not authenticated with Google Cloud. Please run 'gcloud auth login' to authenticate."
    fi
}

# Function to get the service account credentials
get_service_account_credentials() {
    echo "Getting service account credentials..."
    sudo gcloud iam service-accounts keys create "${credentials_location}" --iam-account=bacalhau-buildkite-sa@bacalhau-infra.iam.gserviceaccount.com
}

# Function to update the Buildkite environment hook
update_buildkite_environment_hook() {
    local hook_file="/etc/buildkite-agent/hooks/environment"
    if [[ ! -f "${hook_file}" ]]; then
        echo "Warning: ${hook_file} does not exist. Creating it."
        sudo touch "${hook_file}"
        sudo chmod 755 "${hook_file}"
    fi

    if ! grep -q "GOOGLE_APPLICATION_CREDENTIALS" "${hook_file}"; then
        echo "Updating ${hook_file} with GOOGLE_APPLICATION_CREDENTIALS..."
        echo "export GOOGLE_APPLICATION_CREDENTIALS=${credentials_location}" | sudo tee -a "${hook_file}"
    else
        echo "GOOGLE_APPLICATION_CREDENTIALS is already set in ${hook_file}."
    fi
}

# Main script execution
main() {
    if ! command -v gcloud &> /dev/null; then
        install_gcloud
    fi

    check_gcloud_authentication

    if [[ ! -f "${credentials_location}" ]]; then
        debug_msg "Creating GOOGLE_APPLICATION_CREDENTIALS file at ${credentials_location}"
        get_service_account_credentials
    else
        debug_msg "GOOGLE_APPLICATION_CREDENTIALS file already exists at ${credentials_location}"
    fi

    if [[ ! -f "${credentials_location}" ]]; then
        critical_error "Failed to create GOOGLE_APPLICATION_CREDENTIALS file at ${credentials_location}"
    fi

    update_buildkite_environment_hook

    debug_msg "GOOGLE_APPLICATION_CREDENTIALS set to ${credentials_location}"
}

main
