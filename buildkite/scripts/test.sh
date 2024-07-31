#!/bin/bash

export LOG_LEVEL=DEBUG
export TEST_BUILD_TAGS=$1
export TEST_PARALLEL_PACKAGES=4

make build-webui
make test-and-report
