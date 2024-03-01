#!/usr/bin/env bash
set -xeuo pipefail
IFS=$'\n\t'

if [[ "${BRANCH}" =~ "refs/tags" ]]; then
   TAG=$(echo "${BRANCH}" | sed 's:refs/tags/::')
   TARGET="\"tag\": \"${TAG}\""
else
   TARGET="\"branch\": \"${BRANCH}\""
fi

curl --fail -X POST --header "Content-Type: application/json" --header "Circle-Token: ${CIRCLE_TOKEN}" -d "{
 \"parameters\": {
    \"GHA_Action\": \"trigger_pipeline\"
 },
 ${TARGET}
}" https://circleci.com/api/v2/project/gh/bacalhau-project/bacalhau/pipeline
