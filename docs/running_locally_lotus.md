# Running filecoin lotus local-network alongside the 'devstack'

This guide will help you to run `devstack` alongside lotus filecoin nodes using local network with 2KiB sectors.


## Pre-requisites

 * x86_64 of ARM64 architecture
    * Ubuntu 20.0+ has most often been used for development and testing
 * Go >= 1.18
 * [Docker Engine](https://docs.docker.com/get-docker/)
 * (Optional) A build of the [latest Bacalhau release](https://github.com/filecoin-project/bacalhau/releases/)

## (Optional) Building Bacalhau from source

```bash
sudo apt-get update && sudo apt-get install -y make gcc zip
sudo snap install go --classic
wget https://github.com/filecoin-project/bacalhau/archive/refs/heads/main.zip
unzip main.zip
cd bacalhau-main
go build
```

## Start the devstack using make

```bash
## Run devstack along with filecoin lotus locally
make devstack-lotus
```


Once everything has started up - you will see output like the following:

```bash
Starting sync wait
retry 13 out of 30 ...
Starting sync wait
retry 14 out of 30 ...
Starting sync wait
retry 15 out of 30 ...
Starting sync wait
lotus is healthy
LOTUS_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiLCJzaWduIiwiYWRtaW4iXX0.GhOSV-JjyCeoelCQPfouMiyC-3mmTkVeSoBp5UNX9iI
go run . devstack
10:57:12.163 | INF devstack/devstack.go:324 > Devstack is ready!
10:57:12.163 | INF devstack/devstack.go:325 > To use the devstack, run the following commands in your shell:
10:57:12.163 | INF devstack/devstack.go:326 >
export BACALHAU_IPFS_SWARM_ADDRESSES=/ip4/127.0.0.1/tcp/34609/p2p/QmZNKKMSzWqe8VWkdrwu4nkfXidQ6sHBj4A32SdxW4bj2M
export BACALHAU_API_HOST=0.0.0.0
export BACALHAU_API_PORT=38141
```

The above command will run these make targets in sequence:
`lotus-run`: runs `bacalhau-lotus-image` docker image with the container name `lotus`. Please note that the local directory `testdata` is mounted under `/home/lotus_user/testdata` so you can copy any test files required for the tests.
`lotus-health`: runs health check against the above docker container, has retry mechanism with 5 min timeout.

The command also prints lotus API token in case you want to interact with the [API](https://lotus.filecoin.io/reference/basics/api-access/).

### Run integration test against lotus
```bash
go test github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus -count=1
```

## Lotus build and usage
```bash
## Build lotus image
lotus-image-build
## Build lotus image without cache
lotus-image-rebuild
## Run lotus image
lotus-run
## Run healthcheck
lotus-health
## Bash into the container
lotus-bash
## Print lotus API token
lotus-token
## Print lotus miner API token
lotus-miner-token
## Tail lotus logs
lotus-log
## Clean lotus container
lotus-clean
```