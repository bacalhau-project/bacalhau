#!/usr/bin/env bash

# a script that will upload a CID to nodes in a terraform workspace
# it queries the gcloud CLI for IP addresses
# and then does an "ipfs --api X add X" for each node
set -euo pipefail
IFS=$'\n\t'

export WORKSPACE=${1:-""}
export LOCALPATH=${2:-""}

if [ -z "$WORKSPACE" ]; then
    echo "Usage: $0 <workspace> <local-path>"
    exit 1
fi

if [ -z "$LOCALPATH" ]; then
    echo "Usage: $0 <workspace> <local-path>"
    exit 1
fi

if [ ! -e "$LOCALPATH" ]; then
    echo "$LOCALPATH file or directory not found"
    exit 1
fi

filename=$(basename $LOCALPATH)

for name in $(gcloud compute instances list --format="value(name)" --filter="name~$WORKSPACE"); do
    echo "Uploading $filename to $name"
    gcloud compute scp --recurse $LOCALPATH $name:$filename
    gcloud compute ssh $name -- ipfs --api=/ip4/127.0.0.1/tcp/5001 add -r $filename
    gcloud compute ssh $name -- rm -rf $filename
done
