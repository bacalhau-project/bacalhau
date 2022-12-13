#!/bin/sh
set -euo pipefail
IFS=$'\n\t'
SUDO='' # detect if not root...


echo "CLI params: $1"


if [[ ! -e /var/run/docker.sock ]]; then
  echo "ERROR: no docker socket mounted into the container. Please re-run with:"
  echo
  echo "docker run -dit -v /var/run/docker.sock:/var/run/docker.sock --publish 1234:1234 bacalhauproject/bacalhau:dev"
  return 1
fi

if [[ "${1:-}" == "devstack"  ]]; then
  LOG_LEVEL=debug bacalhau \
    --api-host ${BACALHAU_API_HOST} \
    --api-port ${BACALHAU_API_PORT} \
    devstack
  exit 0
fi

if [[ -z "${IPFS_CONNECT:-}"  ]]; then
  # if IPFS_CONNECT isn't set, ask the docker-compose ipfs container that's starting at the same time
  if ! docker inspect bacalhau-ipfs > /dev/null ; then
    echo "ERROR: IPFS_CONNECT environment variable not set, and no bacalhau-ipfs container detected"
    exit 1
  fi

  echo "auto-detecting ipfs connection address from bacalhau_ipfs container (set IPFS_CONNECT)"
  # if the user doesn't tell us how to connect to ipfs, then try the quckstart container
  # TODO: can't use loopback in this case...
  IPFS_CONNECT=$(docker exec -it bacalhau-ipfs ipfs id --format="/ip4/127.0.0.1/tcp/5001/p2p/<id>")
fi

echo "starting Bacalhau using IPFS_CONNECT=${IPFS_CONNECT}"

LOG_LEVEL=debug bacalhau serve \
  --ipfs-connect ${IPFS_CONNECT}