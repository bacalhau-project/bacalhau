#!/bin/bash
set -xeuo pipefail
docker build -t gibbonsdatascience/entrypoint-cat:0.0.1 .
docker push gibbonsdatascience/entrypoint-cat:0.0.1
