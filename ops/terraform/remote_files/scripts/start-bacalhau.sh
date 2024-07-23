#!/bin/bash
# shellcheck disable=SC1091,SC2312
set -euo pipefail
IFS=$'\n\t'

# import the terraform vars
source /terraform_node/variables

# mount the disk - wait for /dev/sdb to exist
# NB: do not reformat the disk if we can't mount it, unlike the initial
# install/upgrade script!
while [[ ! -e /dev/sdb ]]; do
  sleep 1
  echo "waiting for /dev/sdb to exist"
done
# mount /dev/sdb at /data
mkdir -p /data
mount /dev/sdb /data || true

# import the secrets
source /data/secrets.sh

BACALHAU_PROBE_EXEC='/terraform_node/apply-http-allowlist.sh'
TRUSTED_CLIENT_IDS="\
1df7b01ed77ca81bb6d6f06f6cbcd76a6a9e450d175dfac1e4ba70494fddd576,\
b43517b5449d383ab00ca1d2b1c558d710ba79f51c800fbf4c35ed4d0198aec5"

  # nats related config as set as env vars in main.tf and no need to pass them to serve command
bacalhau serve \
  --node-type "${BACALHAU_NODE_TYPE}" \
  --job-selection-data-locality anywhere \
  --job-selection-accept-networked \
  --job-selection-probe-exec "${BACALHAU_PROBE_EXEC}" \
  --max-job-execution-timeout '60m' \
  --job-execution-timeout-bypass-client-id="${TRUSTED_CLIENT_IDS}" \
  --ipfs-connect /ip4/127.0.0.1/tcp/5001 \
  --api-port 1234 \
  --web-ui="${BACALHAU_NODE_WEBUI}" \
  --web-ui-port 80 \
  --labels owner=bacalhau \
  --requester-job-translation-enabled \
  --default-publisher local \
  --local-publisher-address "${BACALHAU_LOCAL_PUBLISHER_ADDRESS}"
