#!/bin/bash

export PEER_TOKEN=""
ip -4 -j address > /home/ubuntu/health_check/index.html
while [[ -z "$PEER_TOKEN" ]]; do 
    PEER_TOKEN=$(journalctl --unit=bacalhau.service -n 100 --no-pager | sed -En 's/.*?\/ip4\/.*?\/tcp\/0\/p2p\/(.*)/\1/p')
done

echo "--peer /ip4/0.0.0.0/tcp/0/p2p/$PEER_TOKEN" > /home/ubuntu/health_check/peer_token.html