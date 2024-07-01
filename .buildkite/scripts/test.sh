#!/usr/bin/env bash

set -euo pipefail

# # Check if $_FLOX_ACTIVE_ENVIRONMENTS is set and equal to the desired environment
# if [ -z "${FLOX_ENV_DESCRIPTION:-}" ] || [ "$FLOX_ENV_DESCRIPTION" != "$flox_env" ]; then
#   echo "Activating flox environment: $flox_env"

#   # shellcheck disable=SC1090,SC1091,SC2312
#   . <(FLOX_DISABLE_METRICS=true SHELL=$(command -v bash) flox activate -r aronchick/bacalhau -t;)
# else
#   echo "Flox environment $flox_env is already active"
# fi

. <(FLOX_DISABLE_METRICS=true SHELL=$(command -v bash) flox activate -vv -r aronchick/bacalhau -t;)


# Set up environment variables
export LOG_LEVEL='debug'
export TEST_BUILD_TAGS="${BUILD_TAGS}"
export TEST_PARALLEL_PACKAGES='4'
export GOMAXPROCS="2"

# Run tests
just test-and-report
