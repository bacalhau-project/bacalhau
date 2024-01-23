#!/usr/bin/env bash
curl --request POST \
  --url https://circleci.com/api/v2/project/gh/bacalhau-project/bacalhau/pipeline \
  --header "\"Circle-Token\": \"${1}\"" \
  --header 'content-type: application/json' \
  --data '{"parameters":{"run_workflow_lint": true }}'