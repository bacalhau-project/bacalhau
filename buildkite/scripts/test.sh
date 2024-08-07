#!/bin/bash

export LOG_LEVEL=DEBUG
export TEST_BUILD_TAGS=$1
export TEST_PARALLEL_PACKAGES=8
export BACALHAU_ENVIRONMENT=test
export AWS_ACCESS_KEY_ID=$(buildkite-agent secret get AWS_ACCESS_KEY_ID)
export AWS_SECRET_ACCESS_KEY=$(buildkite-agent secret get AWS_SECRET_ACCESS_KEY
export AWS_REGION=eu-west-1

make build-webui
make test-and-report
