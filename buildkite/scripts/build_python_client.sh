#!/bin/bash

set -e

setup_environment_variables() {
  export PYPI_TOKEN=$(buildkite-agent secret get PYPI_TOKEN)
  export TEST_PYPI_TOKEN=$(buildkite-agent secret get TEST_PYPI_TOKEN)
  export RELEASE_PYTHON_PACKAGES=1
}

download_swagger() {
  cd docs
  rm -rf swagger.json
  buildkite-agent artifact download "swagger.json" . --build $BUILDKITE_BUILD_ID
  cd ..
}

build_python_apiclient() {
  make build-python-apiclient
}

publish_python_apiclient() {
  make release-python-apiclient
}


main () {
  setup_environment_variables
  download_swagger
  build_python_apiclient

  if [ -n "$BUILDKITE_TAG" ]; then
    publish_python_apiclient
  fi
}

main
