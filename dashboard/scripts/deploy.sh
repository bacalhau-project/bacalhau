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
  docker build --platform linux/amd64 -t $IMAGE_API -f Dockerfile.dashboard .
  docker save $IMAGE_API | bzip2 | gcloud compute ssh dashboard-vm-default-0 -- sudo docker load
  echo $IMAGE_API
}

function build:frontend() {
  docker build --platform linux/amd64 -t $IMAGE_FRONTEND dashboard/frontend
  docker save $IMAGE_FRONTEND | bzip2 | gcloud compute ssh dashboard-vm-default-0 -- sudo docker load
  echo $IMAGE_FRONTEND
}

function restart() {
  gcloud compute ssh dashboard-vm-default-0 -- cd /data/dashboard && sudo docker-compose stop
  gcloud compute ssh dashboard-vm-default-0 -- cd /data/dashboard && sudo IMAGE_FRONTEND=$IMAGE_FRONTEND IMAGE_API=$IMAGE_API docker-compose up -d
}

eval "$@"