#!/bin/bash

set -e

setup_environment_variables() {
  export PYPI_TOKEN=$(buildkite-agent secret get PYPI_TOKEN)
}

generate_swagger() {
  local path_to_project=$(git rev-parse --show-toplevel)
  local swagger_dir="${path_to_project}/pkg/swagger"
  local webui_path="${path_to_project}/webui/lib/api/schema"
  local docs_path="${path_to_project}/docs"

  cd "${path_to_project}" || exit

  swag init \
    --outputTypes "go,json" \
    --parseDependency \
    --parseInternal \
    --parseDepth 1 \
    -g "./pkg/publicapi/server.go" \
    --overridesFile .swaggo \
    --output "${swagger_dir}"

  echo "swagger.json generated - moving from ${swagger_dir} to ${webui_path}"

  # See if the path exists, if not create it
  if [[ ! -d "${webui_path}" ]]; then
      mkdir -p "${webui_path}"
  fi

  cp "${swagger_dir}/swagger.json" "${webui_path}"
  cp "${webui_path}/swagger.json" "${docs_path}"

  echo "swagger.json copied to ${webui_path}/swagger.json"
  echo "swagger.json copied to ${docs_path}/swagger.json"
}

build_python_apiclient() {
  make build-python-apiclient 
}

publish_python_apiclient() {
  make release-python-apiclient
}


main () {
  setup_environment_variables
  generate_swagger
  build_python_apiclient

  if [-n "$BUILDKITE_TAG" ]; then
    export RELEASE_PYTHON_PACKAGES=1
    publish_python_apiclient
  else
    echo "Buildkite Tag Not Found"
  fi
}

main
