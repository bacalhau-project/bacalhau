#!/usr/bin/env bash
PATH_TO_PROJECT_ROOT="${PWD}/../.."
cd "${PATH_TO_PROJECT_ROOT}" || exit

echo "Currently executing in ${PWD}"
swag init \
  --outputTypes "go,json" \
  --parseDependency \
  --parseInternal \
  --parseDepth 1 \
  -g "./pkg/publicapi/server.go" \
  --overridesFile .swaggo \
  --output "${PATH_TO_PROJECT_ROOT}/pkg/publicapi/swagger"

PUBLIC_PATH="./webui/public/swagger"
echo "swagger.json generated - moving to ${PUBLIC_PATH}"

# See if the path exists, if not create it
if [[ ! -d "${PUBLIC_PATH}" ]]; then
  mkdir -p "${PUBLIC_PATH}"
fi
mv "${PWD}/docs/swagger.json" "${PUBLIC_PATH}"
