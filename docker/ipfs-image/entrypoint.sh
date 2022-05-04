#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

export BACALHAU_FUSE_MOUNT=${BACALHAU_FUSE_MOUNT:-/ipfs}

mkdir -p "$BACALHAU_FUSE_MOUNT/data"
mkdir -p "$BACALHAU_FUSE_MOUNT/ipns"

ipfs init

if [[ -n "$BACALHAU_DELETE_BOOTSTRAP_ADDRESSES" ]]; then

fi

if [[ -n "$BACALHAU_DISABLE_MDNS_DISCOVERY" ]]; then
   ipfs config Discovery.MDNS.Enabled --json false
fi

if [[ -n "$BACALHAU_DELETE_BOOTSTRAP_ADDRESSES" ]]; then
   ipfs bootstrap rm --all
fi

for peerAddress in ${BACALHAU_IPFS_PEER_ADDRESSES//,/ }
do
  ipfs bootstrap add $peerAddress
done

ipfs daemon --mount --mount-ipfs "$BACALHAU_FUSE_MOUNT/data" --mount-ipns "$BACALHAU_FUSE_MOUNT/ipns"