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

function getMultiaddress() {
  echo -n "/ip4/${1}/tcp/${BACALHAU_PORT}/p2p/${2}"
}

# we start with none as the default ("none" prevents the node connecting to our default bootstrap list)
export CONNECT_PEER="none"

# if we are node0 then we do not connect to anything
if [[ "${TERRAFORM_NODE_INDEX}" != "0" ]]; then
  # if we are in unsafe mode - then we connect to a single node and it's ID
  # is pre-determined by the $BACALHAU_NODE0_UNSAFE_ID variable
  if [[ -n "${BACALHAU_UNSAFE_CLUSTER}" ]]; then
    export UNSAFE_NODE0_ID="$BACALHAU_NODE_ID_0"
    if [[ -z "$UNSAFE_NODE0_ID" ]]; then
      export UNSAFE_NODE0_ID="$BACALHAU_NODE0_UNSAFE_ID"
    fi
    export CONNECT_PEER=$(getMultiaddress "$TERRAFORM_NODE0_IP" "$UNSAFE_NODE0_ID")
  # otherwise we will construct our connect string based on
  # what node index we are
  else
    # we are > node0 so we can connect to node0
    export CONNECT_PEER=$(getMultiaddress "$TERRAFORM_NODE0_IP" "$BACALHAU_NODE_ID_0")
    # we are > node1 so we can also connect to node1
    if [[ "${TERRAFORM_NODE_INDEX}" -ge "2" ]]; then
      export CONNECT_PEER="$CONNECT_PEER,$(getMultiaddress "$TERRAFORM_NODE1_IP" "$BACALHAU_NODE_ID_1")"
    fi
    # we are > node2 so we can also connect to node2
    if [[ "${TERRAFORM_NODE_INDEX}" -ge "3" ]]; then
      export CONNECT_PEER="$CONNECT_PEER,$(getMultiaddress "$TERRAFORM_NODE2_IP" "$BACALHAU_NODE_ID_2")"
    fi
  fi
fi

BACALHAU_PROBE_EXEC='/terraform_node/apply-http-allowlist.sh'

TRUSTED_CLIENT_IDS="\
1df7b01ed77ca81bb6d6f06f6cbcd76a6a9e450d175dfac1e4ba70494fddd576,\
b43517b5449d383ab00ca1d2b1c558d710ba79f51c800fbf4c35ed4d0198aec5"

bacalhau serve \
  --node-type requester,compute \
  --job-selection-data-locality anywhere \
  --job-selection-accept-networked \
  --job-selection-probe-exec "${BACALHAU_PROBE_EXEC}" \
  --max-job-execution-timeout '60m' \
  --job-execution-timeout-bypass-client-id="${TRUSTED_CLIENT_IDS}" \
  --ipfs-swarm-addrs "" \
  --ipfs-connect /ip4/127.0.0.1/tcp/5001 \
  --swarm-port "${BACALHAU_PORT}" \
  --api-port 1234 \
  --peer "${CONNECT_PEER}" \
  --private-internal-ipfs=false \
  --labels owner=bacalhau
