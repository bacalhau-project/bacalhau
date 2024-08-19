#!/usr/bin/env bash

set -x

# NB(forrest/udit): this step needs to be done before linting as without it the code doesn't compile since webuid/build DNE.
make build-webui

# NB(forrest/udit): linting cannot be done by pre-commit because it doesn't work...
make lint

# TODO(forrest/udit): deprecate pre-commit and replace it with the individual steps required to validate the code.
# e.g. modtidy check, credentials check, go fmt, test file header validation.
pre-commit run --show-diff-on-failure --color=always --all-files

