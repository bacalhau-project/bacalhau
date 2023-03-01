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

files_without_header=$(grep --include '*_test.go' -lR 'func Test[A-Z].*(t \*testing.T' ./* | xargs grep --files-without-match -e '//go:build integration' -e '//go:build unit || !integration' --)

if [[ -n "${files_without_header}"  ]]; then
  printf "Test files missing '//go:build integration' or '//go:build unit || !integration':\n%s\n" "${files_without_header}"
  exit 1
fi
