#!/bin/bash

set -euo pipefail

# Default values
SKIP_COMPILE=false
SPECIFIC_TEST=""
COMPILE_ONLY=false
NO_FLAGS=true  # New flag to track if any options were provided

# Function to display usage
usage() {
    echo "Usage: ${0} [-s] [-t TEST_NAME] [-c]"
    echo "  -s: Skip compilation"
    echo "  -t TEST_NAME: Run specific test by name"
    echo "  -c: Compile only (don't run tests)"
    echo "  -h: Show this help message"
    exit 1
}

# Parse command line arguments
while getopts ":st:ch" opt; do
    NO_FLAGS=false  # Set to false since we received at least one option
    # shellcheck disable=SC2249
    case ${opt} in
        s) SKIP_COMPILE=true ;;
        t) SPECIFIC_TEST="${OPTARG}" ;;
        c) COMPILE_ONLY=true ;;
        h) usage ;;
        \?) # Invalid option
            echo "Error: Invalid option -${OPTARG}"
            usage
            ;;
        :) # Option missing required argument
            echo "Error: Option -${OPTARG} requires an argument"
            usage
            ;;
    esac
done

# Compilation function
compile_binary() {
    # cd to main repo
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
}

# Run tests function
run_tests() {
    if [[ -n "${SPECIFIC_TEST}" ]]; then
        echo "Running specific test: ${SPECIFIC_TEST}"
        go test -v -run "${SPECIFIC_TEST}"
    else
        echo "Running all tests"
        go test -v -count=1 ./...
    fi
}

# Main execution logic
if ${NO_FLAGS}; then
    # No flags provided - do everything
    cd ..
    compile_binary
    cd test_integration || exit
    run_tests
else
    # Flags were provided - follow the flag logic
    if ! ${SKIP_COMPILE}; then
        cd ..
        compile_binary
        cd test_integration || exit
    fi

    if ! ${COMPILE_ONLY}; then
        run_tests
    fi
fi
