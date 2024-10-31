#!/bin/bash

set -euo pipefail

# Setup cleanup trap
cleanup() {
    echo "Cleaning up..."
    rm -f common_assets/bacalhau_bin
}
trap cleanup EXIT

# cd to main repo
cd ..

# Get GOARCH value and print it for verification
GOARCH=$(go env GOARCH)
echo "Building for architecture: ${GOARCH}"

# Remove old binary if exists
rm -f "bin/linux/${GOARCH}/bacalhau"

# Remove old binary from test_integration directory
rm -f test_integration/common_assets/bacalhau_bin

# Run make build with GOOS=linux
GOOS=linux make build

# Verify the binary exists
if [[ ! -f "bin/linux/${GOARCH}/bacalhau" ]]; then
    echo "Error: Binary was not created at bin/linux/${GOARCH}/bacalhau"
    exit 1
fi

# Copy binary to test assets directory
mkdir -p test_integration/common_assets
cp "bin/linux/${GOARCH}/bacalhau" test_integration/common_assets/bacalhau_bin

cd test_integration || exit

# Run tests
go test -v -count=1 ./...
