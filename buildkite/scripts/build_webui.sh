#!/bin/bash

set -euo pipefail

export DOCKER_USERNAME=$(buildkite-agent secret get DOCKER_USERNAME)
export DOCKER_PASSWORD=$(buildkite-agent secret get DOCKER_PASSWORD)
make build-webui
