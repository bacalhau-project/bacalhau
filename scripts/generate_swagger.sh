#!/usr/bin/env bash
PATH_TO_PROJECT_ROOT="${PWD}/../.."
SWAGGER_DIR="${PATH_TO_PROJECT_ROOT}/pkg/publicapi/swagger"
cd "${PATH_TO_PROJECT_ROOT}" || exit

echo "Currently executing in ${PWD}"
swag init \
  --outputTypes "go,json" \
  --parseDependency \
  --parseInternal \
  --parseDepth 1 \
  -g "./pkg/publicapi/server.go" \
  --overridesFile .swaggo \
  --output "${SWAGGER_DIR}"

PUBLIC_PATH="./webui/public/swagger"
echo "swagger.json generated - moving from ${SWAGGER_DIR} to ${PUBLIC_PATH}"

# See if the path exists, if not create it
if [[ ! -d "${PUBLIC_PATH}" ]]; then
  mkdir -p "${PUBLIC_PATH}"
fi
mv "${SWAGGER_DIR}/swagger.json" "${PUBLIC_PATH}"
