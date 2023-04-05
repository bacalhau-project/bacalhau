#!/bin/bash
set -xeuo pipefail
docker build -t bacalhau-test-entrypoints/default-echo-hello:dev .
