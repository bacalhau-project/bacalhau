#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

export DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

export CI_COMMIT_SHA=$(git rev-parse HEAD)
export DOCKER_REGISTRY=${DOCKER_REGISTRY:=gcr.io}
export GCP_PROJECT_ID=${GCP_PROJECT_ID:=bacalhau-production}
export IMAGE_FRONTEND=$DOCKER_REGISTRY/$GCP_PROJECT_ID/dashboard-frontend:$CI_COMMIT_SHA
export IMAGE_API=$DOCKER_REGISTRY/$GCP_PROJECT_ID/dashboard-api:$CI_COMMIT_SHA

function build:api() {
  docker build -t $IMAGE_API -f Dockerfile.dashboard .
  docker push $IMAGE_API
}

function build:frontend() {
  docker build -t $IMAGE_FRONTEND dashboard/frontend
  docker push $IMAGE_FRONTEND
}

eval "$@"