#!/bin/bash
set -xeuo pipefail

ID=$(bacalhau --api-port=$API_PORT_0 --api-host=localhost docker run --concurrency=3 busybox -- /bin/true)
while true; do
    sleep 0.01
    if [ $(bacalhau --api-port=$API_PORT_0 --api-host=localhost list --wide --id-filter=$ID --output json | jq .\"$ID\".state |grep '"state": 5' |wc -l) -ne 3 ]; then
       echo "JOB $ID FAILED"
   else
       echo "JOB $ID succeeded"
       exit 0
   fi
done
