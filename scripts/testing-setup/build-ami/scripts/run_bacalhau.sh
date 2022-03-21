#!/bin/bash

touch /tmp/peer_string

if [ -x /usr/local/bin/bacalhau ]; then
    /usr/local/bin/bacalhau --jsonrpc-port 54545 serve $(cat /tmp/peer_string)
else
    echo "Bacalhau binary not detected. Exiting."
    exit 1
fi