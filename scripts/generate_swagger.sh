#!/usr/bin/env bash
PATH_TO_PROJECT_ROOT=$(git rev-parse --show-toplevel)
SWAGGER_DIR="${PATH_TO_PROJECT_ROOT}/pkg/swagger"
WEBUI_PATH="${PATH_TO_PROJECT_ROOT}/webui/lib/api/schema"
DOCS_PATH="${PATH_TO_PROJECT_ROOT}/docs"
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

echo "swagger.json generated - moving from ${SWAGGER_DIR} to ${WEBUI_PATH}"

# See if the path exists, if not create it
if [[ ! -d "${WEBUI_PATH}" ]]; then
    mkdir -p "${WEBUI_PATH}"
fi
cp "${SWAGGER_DIR}/swagger.json" "${WEBUI_PATH}"
cp "${WEBUI_PATH}/swagger.json" "${DOCS_PATH}"

echo "swagger.json copied to ${WEBUI_PATH}/swagger.json"
echo "swagger.json copied to ${DOCS_PATH}/swagger.json"
