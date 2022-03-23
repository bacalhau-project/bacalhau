#!/bin/bash

# Setup nginx
echo -n "Starting nginx service... "
systemctl daemon-reload
systemctl enable nginx.service
systemctl start  nginx.service
echo "Done."

# Setup bacalhau
echo -n "Downloading bacalhau... "
curl -s https://bacalhau.org/install.sh | bash -- 
echo "Done."

# This is way more complicated than it should be - I have to write to a tmp file on disk
# because I don't want/know how to pass through an argument from the service to query
# the "first node" and get the peer token.
echo -n "Starting bacalhau service... "
systemctl daemon-reload
systemctl enable bacalhau.service
systemctl start  bacalhau.service
echo "Done."

# Update content
ip -4 -j address > /home/ubuntu/health_check/index.html

PEER_TOKEN=$(journalctl --unit=bacalhau.service -n 100 --no-pager | sed -En 's/.*?\/ip4\/.*?\/tcp\/0\/p2p\/(.*)/\1/p')
echo "--peer /ip4/0.0.0.0/tcp/0/p2p/$PEER_TOKEN" > /home/ubuntu/health_check/peer_token.html
