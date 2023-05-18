#!/bin/bash
set -euxo pipefail
IFS=$'\n\t'

export DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
export ROOT_DIR="${DIR}/../../"

export GOOS=linux
export GOARCH=amd64

export CI_COMMIT_SHA=$(git rev-parse HEAD)
export DOCKER_REGISTRY=${DOCKER_REGISTRY:=gcr.io}
export GCP_PROJECT_ID=${GCP_PROJECT_ID:=bacalhau-production}
export IMAGE_FRONTEND=$DOCKER_REGISTRY/$GCP_PROJECT_ID/dashboard-frontend:$CI_COMMIT_SHA
export IMAGE_API=$DOCKER_REGISTRY/$GCP_PROJECT_ID/dashboard-api:$CI_COMMIT_SHA

function build:api() {
  pushd "${ROOT_DIR}/dashboard/api"
  CGO_ENABLED=0 go build -o dashboard-api

  docker build --platform "${GOOS}/${GOARCH}" -t $IMAGE_API .
  docker save $IMAGE_API | bzip2 | gcloud compute ssh dashboard-vm-default-0 -- sudo docker load
  echo $IMAGE_API
  popd
}

function build:frontend() {
  pushd "${ROOT_DIR}"
  docker build --platform "${GOOS}/${GOARCH}" -t $IMAGE_FRONTEND dashboard/frontend
  docker save $IMAGE_FRONTEND | bzip2 | gcloud compute ssh dashboard-vm-default-0 -- sudo docker load
  echo $IMAGE_FRONTEND
  popd
}

function restart() {
  gcloud compute ssh dashboard-vm-default-0 -- "cd /data/dashboard && sudo docker-compose stop"
  gcloud compute ssh dashboard-vm-default-0 -- "cd /data/dashboard && sudo IMAGE_FRONTEND=$IMAGE_FRONTEND IMAGE_API=$IMAGE_API docker-compose up -d"
}

eval "$@"
