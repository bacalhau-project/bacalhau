#!/bin/bash

set -euo pipefail

# Source docker authentication
source "$(dirname "$0")/docker-auth.sh"
docker_auth

make build-webui
