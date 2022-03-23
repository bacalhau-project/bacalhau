#!/bin/bash

# v0.0.1

# This is way more complicated than it should be - I have to write to a tmp file on disk
# because I don't want/know how to pass through an argument from the service to query
# the "first node" and get the peer token.
if [[ -f /usr/local/bin/bacalhau && -f  /tmp/remote_peer_string ]]; then
    /usr/local/bin/bacalhau --jsonrpc-port 54545 serve $(cat /tmp/remote_peer_string)
else
    echo "Bacalhau binary not detected. Exiting."
    sleep 10
    exit 1
fi 