swag init \
  --outputTypes "json" \
  --parseDependency \
  --parseInternal \
  --parseDepth 1 \
  -g "pkg/publicapi/server.go" \
  --overridesFile .swaggo

# Use an environment variable to determine the path to the swagger.json file
# This is used in the webui to load the swagger.json file
PUBLIC_PATH="../../webui/public/swagger"
echo "swagger.json generated - moving to ${PUBLIC_PATH}"

# See if the path exists, if not create it
if [[ ! -d "${PUBLIC_PATH}" ]]; then
  mkdir -p "${PUBLIC_PATH}"
fi
mv swagger.json "${PUBLIC_PATH}"
