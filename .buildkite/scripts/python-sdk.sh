#!/usr/bin/env bash

set -xe

required_vars=(
  "AWS_ACCOUNT_ID"
  "AWS_USER_NAME"
  "AWS_USER_PASSWORD"
  "AWS_ACCESS_KEY_ID"
  "AWS_SECRET_ACCESS_KEY"
  "AWS_POLICY_ARN"
  "BUCKET_NAME"
  "BUILDKITE_AGENT_TEST_TOKEN"
  "BUILDKITE_S3_ACCESS_KEY_ID"
  "BUILDKITE_S3_SECRET_ACCESS_KEY"
  "BUILDKITE_S3_ACL"
)

# Get the directory of the current script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Construct the path to the .env file
LOCAL_ENV_FILE="$SCRIPT_DIR/.env"

# Check if the .env file exists
if [ -f "$LOCAL_ENV_FILE" ]; then
    # Load the .env file
    set -a
    # shellcheck disable=SC1090
    source "$LOCAL_ENV_FILE"
    set +a
    echo "${LOCAL_ENV_FILE} loaded successfully"
else
    echo "Error: ${LOCAL_ENV_FILE} not found in script directory"
    exit 1
fi


# If ENV_FILE is set, load B64_ENV_FILE from that file
if [[ -n "${ENV_FILE}" ]] && [[ -f "${ENV_FILE}" ]]; then
    env_file_contents=$(cat "${ENV_FILE}")
    B64_ENV_FILE=$(echo "${env_file_contents}" | base64)
fi

# Decode and load environment variables from base64 encoded content
if [ -n "$B64_ENV_FILE" ]; then
    eval "$(echo "${B64_ENV_FILE}" | base64 -d | sed 's/^/export /')"
fi

for var in "${required_vars[@]}"; do
  if [ -z "${!var}" ]; then
    echo "Error: ${var} is not set in .env file"
    exit 1
  fi
done

just build-python-sdk

# If BUILDKITE_AGENT_ACCESS_TOKEN is not set, skip
if [ -z "$BUILDKITE_AGENT_ACCESS_TOKEN" ]; then
    echo "BUILDKITE_AGENT_ACCESS_TOKEN is not set, skipping artifact upload"
    exit 0
fi

buildkite-agent artifact upload "python/dist/*" "s3://$BUCKET_NAME/$BUILDKITE_JOB_ID" \
                                                --job "$BUILDKITE_JOB_ID" \
                                                --agent-access-token "$BUILDKITE_AGENT_TEST_TOKEN"
