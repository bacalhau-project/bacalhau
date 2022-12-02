#!/bin/bash
set -euo pipefail
IFS=$'\n\t'
SUDO='' # detect if not root...

if [[ -z "${IPFS_CONNECT:-}"  ]]; then
  # if bacalhau-ipfs container isn't running, start one
  if ! docker inspect bacalhau-ipfs > /dev/null ; then
    # TODO: can we put the desired version into the bacalhau binary?
    export IPFS_VERSION=$(wget -q -O - https://raw.githubusercontent.com/filecoin-project/bacalhau/main/ops/terraform/production.tfvars | grep --color=never ipfs_version | awk -F'"' '{print $2}')
    #ignoring the repo version - they're all too old atm
    export IPFS_VERSION=latest
    # run IPFS (this should eventually be run in the bacalhau entrypoint)
    # TODO: run container as readonly fs

    echo "Starting the backalhau-ipfs ${IPFS_VERSION} container"
    docker run -d \
        --restart always \
        --name bacalhau-ipfs \
        -v bacalhau_ipfs_staging:/export \
        -v bacalhau_ipfs_data:/data/ipfs \
        -p 4001:4001 \
        -p 4001:4001/udp \
        -p 127.0.0.1:8080:8080 \
        -p 127.0.0.1:5001:5001 \
            ipfs/kubo:${IPFS_VERSION}

    # TODO: how do we test if the image has pulled, and we're started enough to continue?
    sleep 5
  fi

  echo "auto-detecting ipfs connection address from bacalhau_ipfs container (set IPFS_CONNECT)"
  # if the user doesn't tell us how to connect to ipfs, then try the quckstart container
  IPFS_CONNECT=$(docker exec -it bacalhau-ipfs ipfs id --format="/ip4/127.0.0.1/tcp/5001/p2p/<id>")
fi

echo "starting Bacalhau using IPFS_CONNECT=${IPFS_CONNECT}"

LOG_LEVEL=debug bacalhau serve \
  --ipfs-connect ${IPFS_CONNECT}