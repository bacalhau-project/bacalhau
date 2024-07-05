#!/usr/bin/env bash

# Exit immediately if a command exits with a non-zero status
set -e

# Set Go environment variables
export GOARCH=amd64
export GOOS=linux

# Define cache directory
CACHE_DIR="/root/.cache/golangci-lint"

# Create cache directory if it doesn't exist
mkdir -p "${CACHE_DIR}"

# Find Go packages, excluding vendor and test files
IFS=$'\n' # Use newline as delimiter
GO_PACKAGES=()

# Execute go list and check its exit status
if ! go list ./... > "${CACHE_DIR}/go_list_output.txt" 2>&1; then
    echo "Error listing Go packages. Exiting."
    exit 1
fi

# Read from the temporary file instead of directly from the process substitution
while IFS= read -r pkg; do
    if [[ ! "$pkg" =~ /vendor/ && ! "$pkg" =~ /test/ ]]; then
        GO_PACKAGES+=("$pkg")
    fi
done < "${CACHE_DIR}/go_list_output.txt"

# Optionally, remove the temporary file
rm "${CACHE_DIR}/go_list_output.txt"

# Check if Go packages were found
if [ ${#GO_PACKAGES[@]} -eq 0 ]; then
    echo "No Go packages found. Exiting."
    exit 0
fi

# Print the packages that will be linted
echo "Linting the following packages:"
echo "${GO_PACKAGES[*]}"

# Run golangci-lint with optimizations

golangci-lint run \
    --allow-parallel-runners \
    --timeout 10m \
    --config=.golangci.yml \
    --verbose \
    --fast \
    --cache-dir="$CACHE_DIR" \
    "${GO_PACKAGES[*]}" # Ignore: shellcheck disable=SC2128

# Check the exit status
EXIT_STATUS=$?

# Print completion message
if [[ ${EXIT_STATUS} -eq 0 ]]; then
    echo "Linting completed successfully."
else
    echo "Linting failed with exit status ${EXIT_STATUS}."
fi

exit "${EXIT_STATUS}"
