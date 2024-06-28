#!/usr/bin/env bash

# Exit on error. Append || true if you expect an error.
set -o errexit
# Exit on error inside any functions or subshells.
set -o errtrace
# Do not allow use of undefined vars. Use ${VAR:-} to use an undefined VAR
set -o nounset
# Catch the error in case mysqldump fails (but gzip succeeds) in `mysqldump |gzip`
set -o pipefail

# Function to display critical error messages and exit
function critical_error {
    echo "Critical error: $1"
    exit 1
}

# Function to check if the user is authenticated with Google Cloud
check_gcloud_authentication() {
    if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q .; then
        critical_error "User is not authenticated with Google Cloud. Please run 'gcloud auth login' to authenticate."
    fi
}

# Function to get the service account credentials
get_service_account_credentials() {
    local destination=$1
    local service_account_name=$2
    echo "Getting service account credentials..."
    sudo gcloud iam service-accounts keys create "${destination}" --iam-account="${service_account_name}"
}

# Main script execution
main() {
    if [[ -z "$1" || -z "$2" ]]; then
        critical_error "Usage: $0 <destination> <service_account_name>"
    fi

    local destination=$1
    local service_account_name=$2

    check_gcloud_authentication
    get_service_account_credentials "${destination}" "${service_account_name}"
    echo "Credentials file created at ${destination}"
}

main "$@"
