#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

export BACALHAU_IPFS_SWARM_ANNOUNCE_IP=${BACALHAU_IPFS_SWARM_ANNOUNCE_IP:-"127.0.0.1"}
export BACALHAU_IPFS_FUSE_MOUNT=${BACALHAU_IPFS_FUSE_MOUNT:-"/ipfs_mount"}

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

ipfs init --profile test
ipfs bootstrap rm --all
ipfs config AutoNAT.ServiceMode "disabled"
ipfs config Addresses.Gateway "/ip4/127.0.0.1/tcp/$BACALHAU_IPFS_PORT_GATEWAY"
ipfs config Addresses.API "/ip4/127.0.0.1/tcp/$BACALHAU_IPFS_PORT_API"
ipfs config Addresses.Swarm --json "[\"/ip4/0.0.0.0/tcp/$BACALHAU_IPFS_PORT_SWARM\"]"
ipfs config Addresses.Announce --json "[\"/ip4/$BACALHAU_IPFS_SWARM_ANNOUNCE_IP/tcp/$BACALHAU_IPFS_PORT_SWARM\"]"
ipfs config Swarm.EnableHolePunching --bool false
ipfs config Swarm.DisableNatPortMap --bool true
ipfs config Swarm.RelayClient.Enabled --bool false
ipfs config Swarm.RelayService.Enabled --bool false
ipfs config Swarm.Transports.Network.Relay --bool false
ipfs config Swarm.ConnMgr.Type "none"
#ipfs config Routing.Type "none"
ipfs config Discovery.MDNS.Enabled --json false

peerAddresses=$(echo $BACALHAU_IPFS_PEER_ADDRESSES | tr "," "\n")
for peerAddress in $peerAddresses
do
  echo "add bootstrap address $peerAddress"
  ipfs bootstrap add $peerAddress
done

ipfs daemon --mount --mount-ipfs "$BACALHAU_IPFS_FUSE_MOUNT/data" --mount-ipns "$BACALHAU_IPFS_FUSE_MOUNT/ipns"
