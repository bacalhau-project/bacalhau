#!/bin/bash

# Set variables
ROOT_CA_CERT="generated_assets/bacalhau_test_root_ca.crt"
ROOT_CA_KEY="generated_assets/bacalhau_test_root_ca.key"
DAYS_VALID=1825  # 5 years

# Organization name and country (same as before)
ORG_NAME="Bacalhau"
COUNTRY="US"

# Check if the input argument is provided
if [[ -z "$1" ]]; then
    echo "Error: Please provide a string for the Common Name and Subject Alternative Names."
    exit 1
fi

COMMON_NAME="$1"
OUTPUT_CERT="generated_assets/${COMMON_NAME}.crt"
OUTPUT_KEY="generated_assets/${COMMON_NAME}.key"
CSR_PATH="generated_assets/${COMMON_NAME}.csr"
CNF_PATH="generated_assets/${COMMON_NAME}.cnf"

# Check if the files already exist
if [[ -f "${OUTPUT_CERT}" ]] || [[ -f "${OUTPUT_KEY}" ]]; then
    echo "Error: One or both of the following files already exist:"
    [[ -f "${OUTPUT_CERT}" ]] && echo " - ${OUTPUT_CERT}"
    [[ -f "${OUTPUT_KEY}" ]] && echo " - ${OUTPUT_KEY}"
    echo "Please remove or rename the existing files before running this script."
    exit 1
fi

# Generate a private key for the new certificate
echo "Generating certificate signed by the root CA..."
openssl genpkey -algorithm RSA -out "${OUTPUT_KEY}" -pkeyopt rsa_keygen_bits:4096

# Create an OpenSSL configuration file for the SAN
cat > "${CNF_PATH}" <<EOF
[ req ]
default_bits       = 4096
distinguished_name = req_distinguished_name
req_extensions     = v3_req
prompt             = no

[ req_distinguished_name ]
CN = ${COMMON_NAME}
O = ${ORG_NAME}
C = ${COUNTRY}

[ v3_req ]
keyUsage = critical, digitalSignature, cRLSign, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = ${COMMON_NAME}
EOF

# Generate a certificate signing request (CSR) using the config file
openssl req -new -key "${OUTPUT_KEY}" -out "${CSR_PATH}" -config "${CNF_PATH}"

# Sign the certificate with the root CA
openssl x509 -req -in "${CSR_PATH}" -CA "${ROOT_CA_CERT}" -CAkey "${ROOT_CA_KEY}" \
    -out "${OUTPUT_CERT}" -days "${DAYS_VALID}" -sha256 -extensions v3_req -extfile "${CNF_PATH}"

# Clean up the CSR and config file
rm "${CSR_PATH}" "${CNF_PATH}"

echo "Certificate generated and saved to ${OUTPUT_CERT} and ${OUTPUT_KEY}"

echo "Done!"
