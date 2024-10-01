#!/bin/bash

set -e

setup_environment_variables() {
  export PYPI_TOKEN=$(buildkite-agent secret get PYPI_TOKEN)
  export TEST_PYPI_TOKEN=$(buildkite-agent secret get TEST_PYPI_TOKEN)
  export RELEASE_PYTHON_PACKAGES=1
}

build_python_sdk() {
  make build-python-sdk
}

publish_python_sdk() {
  make release-python-sdk
}

main() {
  setup_environment_variables
  build_python_sdk

  if [ -n "$BUILDKITE_TAG" ]; then
    publish_python_sdk
  fi

}

main
