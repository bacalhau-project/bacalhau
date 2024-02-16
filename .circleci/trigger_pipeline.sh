#!/usr/bin/env bash
curl -X POST --header "Content-Type: application/json" --header "Circle-Token: ${CIRCLE_TOKEN}" -d "{
 \"parameters\": {
    \"GHA_Action\": \"trigger_pipeline\"
 },
 \"branch\": \"${BRANCH}\"
}" https://circleci.com/api/v2/project/gh/bacalhau-project/bacalhau/pipeline
