#!/bin/bash

# a script that will automatically connect to the correct google project
# based on a terraform variables file
# a bit like kubectx but for bacalhau terraform clusters
set -euo pipefail
IFS=$'\n\t'

export WORKSPACE=${1:-""}
export VARIABLES_FILE=${VARIABLES_FILE:-"$WORKSPACE.tfvars"}

if [ -z "$WORKSPACE" ]; then
  echo "Usage: $0 <workspace>"
  exit 1
fi

if [ ! -f "$VARIABLES_FILE" ]; then
  echo "$VARIABLES_FILE file not found"
  exit 1
fi

function get_variable() {
  cat $VARIABLES_FILE | grep "$1" | awk '{print $3}'
}

eval "gcloud config set project $(get_variable gcp_project)"
eval "gcloud config set compute/zone $(get_variable zone)"
eval "terraform workspace select $WORKSPACE"