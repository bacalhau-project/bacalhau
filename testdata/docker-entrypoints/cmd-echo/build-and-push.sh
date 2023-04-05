#!/bin/bash
set -xeuo pipefail
docker build -t gibbonsdatascience/default-echo-hello:0.0.1 .
docker push gibbonsdatascience/default-echo-hello:0.0.1
