#!/bin/bash
set -xeuo pipefail
export DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

export IMAGE=${IMAGE:-binocarlos/bacalhau-ignite-image:latest}

docker build -t $IMAGE $DIR/../docker/ignite-image
docker push $IMAGE
