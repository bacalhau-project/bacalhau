#!/usr/bin/env bash
set -euxo pipefail

# Script to be used with --job-selection-probe-exec that will:
# - reject jobs with --network=Full
# - reject jobs with --domain=... not in our allowlist
# - accept all other jobs

ALLOWLIST=./http-domain-allowlist.txt

TYPE=$(echo "$BACALHAU_JOB_SELECTION_PROBE_DATA" | jq -r '.Job.Tasks[0].Network.Type')
test "$TYPE" = 'HTTP' || test "$TYPE" = 'None'

cd "$(dirname $0)"
MISSING=$(comm -13 \
    <(cat "$ALLOWLIST" | grep -v '#' | sort) \
    <(echo "$BACALHAU_JOB_SELECTION_PROBE_DATA" | jq -r '.Job.Tasks[0].Network.Domains[]' | sort))

test -z "$MISSING"
