#!/bin/bash
set -euo pipefail
IFS=$'\n\t'
SUDO='' # detect if not root...

if [[ -z "${IPFS_CONNECT:-}"  ]]; then
  echo "auto-detecting ipfs connection address from bacalhau_ipfs container (set IPFS_CONNECT)"
  # if the user doesn't tell us how to connect to ipfs, then try the quckstart container
  IPFS_CONNECT=$(docker exec -it bacalhau-ipfs ipfs id --format="/ip4/127.0.0.1/tcp/5001/p2p/<id>")
fi

echo "starting Bacalhau using IPFS_CONNECT=${IPFS_CONNECT}"

LOG_LEVEL=debug bacalhau serve \
  --ipfs-connect ${IPFS_CONNECT}