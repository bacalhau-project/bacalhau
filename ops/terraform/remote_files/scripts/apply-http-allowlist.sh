#!/usr/bin/env bash
set -euo pipefail

# Script to be used with --job-selection-probe-exec that will:
# - reject jobs with --network=Full
# - reject jobs with --domain=... not in our allowlist
# - accept all other jobs

ALLOWLIST=./http-domain-allowlist.txt

TYPE=$(echo "$BACALHAU_JOB_SELECTION_PROBE_DATA" | jq -r '.Job.Tasks[] | .Network.Type')
if ! (test "$TYPE" = 'HTTP' || test "$TYPE" = 'None' || test "$TYPE" = 'Undefined'); then
    echo "only accept jobs using Network.Type of HTTP, Undefined or None" 1>&2
    exit 1
fi

cd "$(dirname $0)"
MISSING=$(comm -13 \
    <(cat "$ALLOWLIST" | grep -v '#' | sort) \
    <(echo "$BACALHAU_JOB_SELECTION_PROBE_DATA" | jq -r '.Job.Tasks[] | .Network.Domains[]?' | sort))

if ! (test -z "$MISSING"); then
    echo "do not accept jobs which require domains $(echo $MISSING). " 1>&2
    echo "see https://github.com/bacalhau-project/bacalhau/blob/main/ops/terraform/remote_files/scripts/http-domain-allowlist.txt for the allowable list" 1>&2
    exit 1
fi
