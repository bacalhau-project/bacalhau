#!/bin/bash
set -euo pipefail
IFS=$'\n\t'
SUDO='' # detect if not root...

# configure
${SUDO} mkdir -p /data/ipfs
#export IPFS_PATH=/data/ipfs
${SUDO} chown $(id -un):$(id -gn) ${IPFS_PATH} # change ownership of ipfs directory
ipfs init

#launch
ipfs daemon

# TODO: need to explain the ports, and what they do so we don't _have_ to use --net host
# heck - what are the ports that you would not want to expose on your laptop at a hacking conf?