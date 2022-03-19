#!/bin/bash

PEER_STRING=$(cat /tmp/PEER_STRING 2> /dev/null)
echo "Worker Peer string: $PEER_STRING"

read -r -d '' bacalhau_service <<- EOM
[Unit]
Description=Bacalhau
[Service]
User=ubuntu
WorkingDirectory=/home/ubuntu/
ExecStart=/usr/local/bin/bacalhau --jsonrpc-port 54545 serve ${PEER_STRING}
Type=simple
TimeoutStopSec=10
Restart=on-failure
RestartSec=5
[Install]
WantedBy=multi-user.target
EOM

echo -n "Downloading bacalhau... "
curl -s https://bacalhau.org/install.sh | bash -- &> /dev/null 
echo "Done."

pkill ipfs 

echo -n "Installing bacalhau service... "
rm -f /etc/systemd/system/bacalhau.service /tmp/bacalhau.service
printf "%s" "$bacalhau_service" > /tmp/bacalhau.service
mv -f /tmp/bacalhau.service /etc/systemd/system/bacalhau.service
sleep 5
systemctl daemon-reload
systemctl enable bacalhau.service
systemctl start  bacalhau.service
echo "Done."

sleep 5

journalctl --unit=bacalhau.service -n 100 --no-pager | sed -En 's/.*?\/ip4\/.*?\/tcp\/0\/p2p\/(.*)/\1/p' > /tmp/bacalhau_peer_token
