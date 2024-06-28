#!/usr/bin/env bash

# Exit on error. Append || true if you expect an error.
set -o errexit
# Exit on error inside any functions or subshells.
set -o errtrace
# Do not allow use of undefined vars. Use ${VAR:-} to use an undefined VAR
set -o nounset
# Catch the error in case mysqldump fails (but gzip succeeds) in `mysqldump |gzip`
set -o pipefail
# Turn on traces, useful while debugging but commented out by default
#set -o xtrace

# Constants
BUILDKITE_INFRA_PROJECT="bacalhau-infra"
GCP_CREDENTIALS_FILE="/etc/bacalhau-buildkite-sa@bacalhau-infra.iam.gserviceaccount.com-credentials.json"
BUILDKITE_CONFIG_FILE="/etc/buildkite-agent/buildkite-agent.cfg"
SERVICE_ACCOUNT_NAME="bacalhau-buildkite-sa@bacalhau-infra.iam.gserviceaccount.com"
SECRET_NAME="build-agent-env-file"

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

# Parse flags into an array (if any)
flags=()
while [[ $# -gt 0 ]]; do
    flags+=("$1")
    shift
done

# Check for --debug flag
if [[ "${flags[*]}" == *"--debug"* ]]; then
    debug_mode=1
    debug_msg "Debug mode is ON."
fi

# Check for --get-credentials flag
if [[ "${flags[*]}" == *"--get-credentials"* ]]; then
    if [[ -z "$2" ]]; then
        critical_error "No destination provided for credentials file"
    fi
    destination=$2
    shift 2
    ./get_gcp_credentials.sh "${destination}" "${SERVICE_ACCOUNT_NAME}"
    exit 0
fi

# Function to add Buildkite GPG key
add_gpg_key() {
  echo "Adding Buildkite GPG key..."
  curl -fsSL https://keys.openpgp.org/vks/v1/by-fingerprint/32A37959C2FA5C3C99EFBC32A79206696452D198 | sudo gpg --batch --yes --dearmor -o /usr/share/keyrings/buildkite-agent-archive-keyring.gpg
}

# Function to add Buildkite repository
add_repository() {
  echo "Adding Buildkite repository..."
  echo "deb [signed-by=/usr/share/keyrings/buildkite-agent-archive-keyring.gpg] https://apt.buildkite.com/buildkite-agent stable main" | sudo tee /etc/apt/sources.list.d/buildkite-agent.list
}

# Function to update package lists and install Buildkite agent
install_buildkite_agent() {
  echo "Updating package lists and installing Buildkite agent..."
  sudo apt-get update && sudo apt-get install -y buildkite-agent
}

# Function to configure Buildkite agent
configure_buildkite_agent() {
  local agent_token="$1"
  echo "Configuring Buildkite agent..."
  sudo sed -i "s/xxx/${agent_token}/g" ${BUILDKITE_CONFIG_FILE}
}

# Function to enable and start Buildkite agent service
start_buildkite_agent() {
  echo "Enabling and starting Buildkite agent service..."
  sudo systemctl enable buildkite-agent && sudo systemctl start buildkite-agent
}

# Function to install Docker
install_docker() {
  echo "Installing Docker..."
  curl -sSL https://get.docker.com/ | sudo bash
}

install_gcloud() {
  echo "Installing the latest gcloud CLI..."
  # Ensure the repository is added only once
  if ! grep -q "^deb .*cloud-sdk main" /etc/apt/sources.list.d/google-cloud-sdk.list; then
    echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
  fi

  # Overwrite the GPG key file if it exists
  curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo gpg --batch --yes --dearmor -o /usr/share/keyrings/cloud.google.gpg
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
    sudo gcloud iam service-accounts keys create "${GCP_CREDENTIALS_FILE}" --iam-account="${SERVICE_ACCOUNT_NAME}"
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
        echo "export GOOGLE_APPLICATION_CREDENTIALS=${GCP_CREDENTIALS_FILE}" | sudo tee -a "${hook_file}"
    else
        echo "GOOGLE_APPLICATION_CREDENTIALS is already set in ${hook_file}."
    fi
}

authenticate_gcloud() {
  echo "Authenticating gcloud CLI..."
  # If the file exists, overwrite it
  if [ -f "${GCP_CREDENTIALS_FILE}" ]; then
    echo "Overwriting ${GCP_CREDENTIALS_FILE}..."
    gcloud auth activate-service-account --key-file="${GCP_CREDENTIALS_FILE}"
  else
    # Service account file doesn't exist, create it
    echo "Creating ${GCP_CREDENTIALS_FILE}..."

    # Make sure the user running this script has permissions
    if [ "$(id -u)" -ne 0 ]; then
      echo "Error: This script must be run as root."
      exit 1
    fi

    # Make sure it has GCP permissions - if not, run gcloud auth login
    if ! gcloud auth list; then
      echo "Error: This script must have GCP permissions."
      exit 1
    fi

    # Get the service account key
    get_service_account_credentials
  fi
  gcloud config set project "${BUILDKITE_INFRA_PROJECT}"
}

run_gcloud_test() {
  echo "Running gcloud test against bacalhau-infra to get all secrets..."
  gcloud secrets versions access latest --secret="${SECRET_NAME}" --project="${BUILDKITE_INFRA_PROJECT}"
}

move_environment_file() {
  echo "Moving the environment file..."
  if [ -f /etc/buildkite-agent/hooks/environment.sample ]; then
    sudo mv /etc/buildkite-agent/hooks/environment.sample /etc/buildkite-agent/hooks/environment
  else
    echo "Warning: /etc/buildkite-agent/hooks/environment.sample does not exist."
  fi
}

# Main script execution
main() {
  if [[ -z "${BUILDKITE_AGENT_TOKEN:-}" ]]; then
    echo "Error: BUILDKITE_AGENT_TOKEN environment variable is not set."
    echo "Please set the BUILDKITE_AGENT_TOKEN environment variable and try again."
    exit 1
  fi

  local agent_token="${BUILDKITE_AGENT_TOKEN}"

  add_gpg_key
  add_repository
  install_buildkite_agent
  configure_buildkite_agent "${agent_token}"
  install_docker
  install_gcloud
  check_gcloud_authentication

  if [[ ! -f "${GCP_CREDENTIALS_FILE}" ]]; then
    debug_msg "Creating GOOGLE_APPLICATION_CREDENTIALS file at ${GCP_CREDENTIALS_FILE}"
    get_service_account_credentials
  else
    debug_msg "GOOGLE_APPLICATION_CREDENTIALS file already exists at ${GCP_CREDENTIALS_FILE}"
  fi

  if [[ ! -f "${GCP_CREDENTIALS_FILE}" ]]; then
    critical_error "Failed to create GOOGLE_APPLICATION_CREDENTIALS file at ${GCP_CREDENTIALS_FILE}"
  fi

  update_buildkite_environment_hook
  authenticate_gcloud
  run_gcloud_test
  move_environment_file
  start_buildkite_agent

  echo "Buildkite agent installation and configuration complete."
}

main "$@"
