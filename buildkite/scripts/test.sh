#!/bin/bash

export LOG_LEVEL=DEBUG
export TEST_BUILD_TAGS=$1
export TEST_PARALLEL_PACKAGES=3
export BACALHAU_ENVIRONMENT=test

make build-webui
make test-and-report
