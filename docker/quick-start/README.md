# Quickstart a test public-cluster Bacalhau node using a container

```
docker build -t bacalhau --target bacalhau .

#ignoring the repo version - they're all too old atm
export IPFS_VERSION=$(wget -q -O - https://raw.githubusercontent.com/filecoin-project/bacalhau/main/ops/terraform/production.tfvars | grep --color=never ipfs_version | awk -F'"' '{print $2}')
export IPFS_VERSION=latest
# run IPFS (this should eventually be run in the bacalhau entrypoint)
# TODO: run container as readonly fs

docker run -d \
    --restart always \
    --name bacalhau-ipfs \
    -v ipfs_staging:/export \
    -v ipfs_data:/data/ipfs \
    -p 4001:4001 \
    -p 4001:4001/udp \
    -p 127.0.0.1:8080:8080 \
    -p 127.0.0.1:5001:5001 \
        ipfs/kubo:${IPFS_VERSION}

docker logs -f bacalhau-ipfs

<<<< mention http://127.0.0.1:5001/webui >>>>

ctrl-c

# check that your ipfs container can get data
docker exec -it bacalhau-ipfs  ipfs cat /ipfs/QmQPeNsJPyVWPFDVHb77w8G42Fvo15z4bG2X8D2GhfbSXc/readme

#export IPFS_CONNECT=$(docker exec -it bacalhau-ipfs  ipfs id | grep #tcp | grep 127.0.0.1 | sed 's/4001/5001/' | sed 's/^[ \t]*//' | sed #'s/,//')

#TODO - : 02:07:28.327 | DBG bacalhau/serve.go:284 > libp2p connecting to: [/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL /ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF /ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3]
': failed to parse multiaddr "\"/ip4/127.0.0.1/tcp/5001/p2p/12D3KooWDN19JJkptojgerCwHYYJcZcz7whuMGsm7Dqd86kutGb3\"\r": must begin with /b3"

# TODO: run container as readonly fs
# TODO: and make a shared /tmp...?
docker run \
    -dit \
    --name bacalhau \
    --restart always \
    --net host \
    --volume bacalhau-data:/data \
    --volume /run/docker.sock:/run/docker.sock \
    --volume /tmp:/tmp \
    --volume $(which docker):$(which docker) \
        bacalhau


docker logs -f bacalhau

ctrl-c

# To see that bacalhau is running your docker job locally, you can run 'docker events` in another terminal, and you should see the container 'create', 'start', 'die', and 'destroy' events for a container with a name starting with 'bacalhau'

docker exec -it \
    --env BACALHAU_API_HOST=127.0.0.1 \
    --env BACALHAU_API_PORT=1234 \
    bacalhau \
     bacalhau docker run ubuntu echo hello

# Interestingly, because your bacalhau node is a part of the bootstrap-cluster, you can get info about your job directly from your node, or via the <<<insert dns..name>>>
<<<<<< I wonder if the devstack turns that off, or if those docs need more detail too>>>>>>

## TODO: can get failures from:
## 07:20:00.35 | INF system/cleanup.go:71 > could not create memory profile error="open /tmp/bacalhau-devstack-mem.prof: permission denied"
## turned out my file was there and owned by my user, not the container root... (because it's made by the client side - and thus clashes with running it locally...)

sven@p1:~/src/ipfs/bacalhau/docker/quick-start$ ../../bacalhau describe 6c9068ec-dff9-4273-b4c1-c160bfa29c57
APIVersion: V1beta1
ClientID: f76a38548c387e85fc8bd2927c9a426949ddae40463f4ade51cb6ef04fcfb298
CreatedAt: "2022-11-24T02:29:12.955258918Z"
Deal:
  Concurrency: 1
ExecutionPlan:


<<<<<mmm, so we hooked up an ipfs-node to bacalhau? how do we know that the job output was stored there?>>>>>


<<< so is my node now able to have  other people's jobs run on it? and how do i know??>>>

## for bacalhau commands that don't need access to your local disk, you can make an alias:
alias bacalhau='docker exec -it \
    --env BACALHAU_API_HOST=127.0.0.1 \
    --env BACALHAU_API_PORT=1234 \
    bacalhau bacalhau'
.......... snip..............
```

## Stop & Remove the containers and volumes

```

docker rm -f bacalhau-ipfs bacalhau
docker volume rm bacalhau-ipfs bacalhau-data
```