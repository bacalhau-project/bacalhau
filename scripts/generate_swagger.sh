#!/usr/bin/env bash
PATH_TO_PROJECT_ROOT=$(git rev-parse --show-toplevel)
SWAGGER_DIR="${PATH_TO_PROJECT_ROOT}/pkg/swagger"
PUBLIC_PATH="${PATH_TO_PROJECT_ROOT}/webui/public/swagger"
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

echo "swagger.json generated - moving from ${SWAGGER_DIR} to ${PUBLIC_PATH}"

# See if the path exists, if not create it
if [[ ! -d "${PUBLIC_PATH}" ]]; then
  mkdir -p "${PUBLIC_PATH}"
fi
mv "${SWAGGER_DIR}/swagger.json" "${PUBLIC_PATH}"

echo "swagger.json also copied to ${PUBLIC_PATH}/swagger.json"
cp "${PUBLIC_PATH}/swagger.json" "${PATH_TO_PROJECT_ROOT}/docs/swagger.json"
