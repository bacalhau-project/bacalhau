#!/usr/bin/env bash 

set -e 

if ! command -v openssl &> /dev/null
then
    echo "openssl is required to create certificates"
    exit 
fi

mkdir ./certs 2> /dev/null
cd ./certs 

cat > dev-server.cnf << EOF
[req]
default_md = sha256
prompt = no
req_extensions = v3_ext
distinguished_name = req_distinguished_name

[req_distinguished_name]
CN = localhost

[v3_ext]
keyUsage = critical,digitalSignature,keyEncipherment
extendedKeyUsage = critical,serverAuth,clientAuth
subjectAltName = DNS:localhost
subjectAltName = IP:0.0.0.0
subjectAltName = IP:127.0.0.1
EOF

function bail() {
    echo "Error: $1"
    exit 1 
}

# Generate a certificate 
openssl req -new -newkey rsa:2048 -keyout dev-ca.key -x509 -sha256 -days 1826  -out dev-ca.crt \
  -subj "/C=US/ST=RandomState/L=RandomLocation /O=Organisation/OU=OrgUnit/CN=localhost/emailAddress=user@local.local" \
  -passout pass:pass 2> /dev/null || bail "failed to create cert"

# Using the server config and a newly generated server key, create a CSR 
openssl genrsa -out dev-server.key 2048 2> /dev/null ||  bail "failed to generate key"
openssl req -new -key dev-server.key -out dev-server.csr -config dev-server.cnf -passin pass:pass 2> /dev/null || bail "failed to create CSR"


# Verify the certificate signing request 
openssl req -noout -text -in dev-server.csr > /dev/null  2> /dev/null || bail "failed to verify CSR"

# Complete the self-signing of our certificate 
openssl x509 -req -in dev-server.csr -CA dev-ca.crt -CAkey dev-ca.key \
  -CAcreateserial -out dev-server.crt -days 365 -sha256 -extfile dev-server.cnf -extensions v3_ext \
  -passin pass:pass 2> /dev/null || bail "failed to sign CSR"


echo "Certificate generated"