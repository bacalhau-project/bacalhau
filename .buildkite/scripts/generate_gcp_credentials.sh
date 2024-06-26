#!/usr/bin/env bash

set -euo pipefail

# Get .env from the same directory this is in
. "$(dirname "$0")/.env"

gcloud iam service-accounts keys create "$(dirname "$0")/${GCP_SERVICE_ACCOUNT_EMAIL}-credentials.json" --iam-account="${GCP_SERVICE_ACCOUNT_EMAIL}"
