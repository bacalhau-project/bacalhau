#!/bin/bash

set -e

setup_environment_variables() {
  export PYPI_TOKEN=$(buildkite-agent secret get PYPI_TOKEN)
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

  if [-n "$BUILDKITE_TAG" ]
    export RELEASE_PYTHON_PACKAGES=1
    publish_python_sdk
  else 
    echo "Buildkite Tag not found"
  fi
}



