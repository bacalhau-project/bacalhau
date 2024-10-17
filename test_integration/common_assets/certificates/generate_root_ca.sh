#!/bin/bash

# Set variables
ROOT_CA_CERT="generated_assets/bacalhau_test_root_ca.crt"
ROOT_CA_KEY="generated_assets/bacalhau_test_root_ca.key"
DAYS_VALID=3650  # 10 years

# Organization name and country
ORG_NAME="Bacalhau"
COUNTRY="US"

# Check if the files already exist
if [[ -f "${ROOT_CA_CERT}" ]] || [[ -f "${ROOT_CA_KEY}" ]]; then
    echo "Error: One or both of the following files already exist:"
    [[ -f "${ROOT_CA_CERT}" ]] && echo " - ${ROOT_CA_CERT}"
    [[ -f "${ROOT_CA_KEY}" ]] && echo " - ${ROOT_CA_KEY}"
    echo "Please remove or rename the existing files before running this script."
    exit 1
fi

# Generate a Root CA
echo "Generating Root CA..."
openssl genpkey -algorithm RSA -out "${ROOT_CA_KEY}" -pkeyopt rsa_keygen_bits:4096
openssl req -x509 -new -nodes -key "${ROOT_CA_KEY}" -sha256 -days "${DAYS_VALID}" \
    -out "${ROOT_CA_CERT}" -subj "/CN=BacalhauTestRootCA/O=${ORG_NAME}/C=${COUNTRY}"

echo "Root CA generated and saved to ${ROOT_CA_CERT} and ${ROOT_CA_KEY}"

echo "Done!"
