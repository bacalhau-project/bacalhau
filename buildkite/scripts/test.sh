#!/bin/bash

set -euo pipefail

# Source docker authentication
source "$(dirname "$0")/docker-auth.sh"
docker_auth

# Function to set environment variables
set_environment_variables() {
    export LOG_LEVEL=DEBUG
    export TEST_BUILD_TAGS=$1
    export TEST_PARALLEL_PACKAGES=8
    export BACALHAU_ENVIRONMENT=test
    export AWS_ACCESS_KEY_ID=$(buildkite-agent secret get AWS_ACCESS_KEY_ID)
    export AWS_SECRET_ACCESS_KEY=$(buildkite-agent secret get AWS_SECRET_ACCESS_KEY)
    export BUILDKITE_ANALYTICS_TOKEN=$(buildkite-agent secret get TESTSUITE_TOKEN)
    export AWS_REGION=eu-west-1
}

# Function to initialize IPFS
initialize_ipfs() {
    ipfs init
    ipfs config Addresses.API /ip4/127.0.0.1/tcp/5001
    ipfs config Addresses.Gateway /ip4/0.0.0.0/tcp/8080
    ipfs daemon --offline &
    export BACALHAU_NODE_IPFS_CONNECT=/ip4/127.0.0.1/tcp/5001
}

# Function to build web UI
build_web_ui() {
    make build-webui
}

# Function to run tests and report
run_tests_and_report() {
    make test-and-report
}

# Function to upload test results
upload_test_results() {
    curl \
      -X POST \
      --fail-with-body \
      --verbose \
      -H "Authorization: Token token=\"$BUILDKITE_ANALYTICS_TOKEN\"" \
      -F "data=@unittests.xml" \
      -F "format=junit" \
      -F "run_env[CI]=buildkite" \
      -F "run_env[key]=$BUILDKITE_BUILD_ID" \
      -F "run_env[number]=$BUILDKITE_BUILD_NUMBER" \
      -F "run_env[job_id]=$BUILDKITE_JOB_ID" \
      -F "run_env[branch]=$BUILDKITE_BRANCH" \
      -F "run_env[commit_sha]=$BUILDKITE_COMMIT" \
      -F "run_env[message]=$BUILDKITE_MESSAGE" \
      -F "run_env[url]=$BUILDKITE_BUILD_URL" \
      https://analytics-api.buildkite.com/v1/uploads
}

# Main function to execute the script
test() {
    set_environment_variables $1
    initialize_ipfs
    build_web_ui
    run_tests_and_report
    TEST_RUN_EXIT_CODE=$?  # Store the exit code for run_tests_and_report. This way we send failure results to buildkite and fail the pipeline
    upload_test_results
    exit $TEST_RUN_EXIT_CODE
}

# Call the main function with the provided test tag
test "$1"
