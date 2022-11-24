#!/bin/bash
set -euo pipefail
IFS=$'\n\t'
SUDO='' # detect if not root...


echo "starting Bacalhau using IPFS_CONNECT=${IPFS_CONNECT}"

LOG_LEVEL=debug bacalhau serve \
  --ipfs-connect ${IPFS_CONNECT}