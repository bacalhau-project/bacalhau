#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

export BACALHAU_FUSE_MOUNT=${BACALHAU_FUSE_MOUNT:-"/ipfs_mount"}

if [[ -z "$BACALHAU_IPFS_PORT_GATEWAY" ]]; then
  echo >&2 "BACALHAU_IPFS_PORT_GATEWAY required"
  exit 1
fi

if [[ -z "$BACALHAU_IPFS_PORT_API" ]]; then
  echo >&2 "BACALHAU_IPFS_PORT_API required"
  exit 1
fi

if [[ -z "$BACALHAU_IPFS_PORT_SWARM" ]]; then
  echo >&2 "BACALHAU_IPFS_PORT_SWARM required"
  exit 1
fi

ipfs init

ipfs config Addresses.Gateway "/ip4/127.0.0.1/tcp/$BACALHAU_IPFS_PORT_GATEWAY"
ipfs config Addresses.API "/ip4/127.0.0.1/tcp/$BACALHAU_IPFS_PORT_API"
ipfs config Addresses.Swarm --json "[\"/ip4/0.0.0.0/tcp/$BACALHAU_IPFS_PORT_SWARM\"]"

if [[ -n "$BACALHAU_DISABLE_MDNS_DISCOVERY" ]]; then
  echo "disabling mdns discovery"
   ipfs config Discovery.MDNS.Enabled --json false
fi

if [[ -n "$BACALHAU_DELETE_BOOTSTRAP_ADDRESSES" ]]; then
  echo "delete default bootstrap addresses"
   ipfs bootstrap rm --all
fi

peerAddresses=$(echo $BACALHAU_IPFS_PEER_ADDRESSES | tr "," "\n")
for peerAddress in $peerAddresses
do
  echo "add bootstrap address $peerAddress"
  ipfs bootstrap add $peerAddress
done

ipfs daemon --mount --mount-ipfs "$BACALHAU_FUSE_MOUNT/data" --mount-ipns "$BACALHAU_FUSE_MOUNT/ipns"
