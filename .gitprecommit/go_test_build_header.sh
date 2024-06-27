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

# Function to find test files
find_test_files() {
  grep --exclude-dir='*vendor*' --include '*_test.go' -lR 'func Test[A-Z].*(t \*testing.T' ./* || {
    echo "Error: Failed to find test files."
    exit 1
  }
}

# Function to check for missing build headers
check_missing_headers() {
  local test_files=("$@")
  local files_without_header=()

  for file in "${test_files[@]}"; do
    if ! grep -q -e '//go:build integration || !unit' -e '//go:build unit || !integration' "$file"; then
      files_without_header+=("$file")
    fi
  done

  if [ ${#files_without_header[@]} -ne 0 ]; then
    printf "%s\n" "${files_without_header[@]}"
    return 1
  fi
}

# Main script execution
main() {
  local test_files
  test_files=$(find_test_files)

  if [[ -n "${test_files}" ]]; then
    IFS=$'\n' read -r -d '' -a test_files_array <<< "$test_files"
    local files_without_header
    files_without_header=$(check_missing_headers "${test_files_array[@]}")

    if [[ -n "${files_without_header}" ]]; then
      printf "Test files missing '//go:build integration || !unit' or '//go:build unit || !integration':\n%s\n" "${files_without_header}"
      exit 1
    else
      echo "All test files have the required build headers."
      exit 0
    fi
  else
    echo "No test files found."
    exit 0
  fi
}

main
