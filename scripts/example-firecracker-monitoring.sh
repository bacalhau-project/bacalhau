#!/bin/bash
set -xeuo pipefail

NAME=boom1
sudo ignite rm -f $NAME || true
sudo ignite run weaveworks/ignite-ubuntu --name $NAME --cpus 2 --memory 1GB --size 6GB --ssh
PID=$(sudo ps auxwwww |grep $(sudo ignite inspect vm $NAME | jq -r .metadata.uid) |grep 'firecracker --api-sock' |awk '{print $2}')

sudo ignite exec $NAME "apt update && apt-get install -y unzip"
sudo ignite exec $NAME "wget https://eforexcel.com/wp/wp-content/uploads/2020/09/5m-Sales-Records.zip"

sudo psrecord $PID --plot outputs/$NAME-$PID.png &
TRACE_PID=$!
echo $NAME has pid $PID, tracing with $TRACE_PID

sudo ignite exec $NAME "unzip 5m-Sales-Records.zip"
sleep 10
for X in {1..10}; do
    sudo ignite exec $NAME "sed 's/Office Supplies/Booze/' '5m Sales Records.csv' -i"
    sleep 2
done
sleep 10
sudo kill $TRACE_PID