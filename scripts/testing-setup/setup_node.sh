#!/bin/bash

echo -n "Downloading bacalhau... "
curl -s https://bacalhau.org/install.sh | bash -- &> /dev/null 
echo "Done."

pkill ipfs
pkill bacalhau

cat $1 > /tmp/peer_string

echo -n "Installing bacalhau service... "
sleep 5
systemctl daemon-reload
systemctl enable bacalhau.service
systemctl start  bacalhau.service
echo "Done."

sleep 5

/usr/local/bin/scripts/update_peer_and_ip.sh