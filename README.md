# Bacalhau - The Filecoin Distributed Computation Framework

## Background

To read more about this project, please read our [Read Me](https://github.com/filecoin-project/bacalhau/wiki)

## Latest

Our most recent demo (2022-02-11):

https://user-images.githubusercontent.com/264658/152514573-b7b115ce-4123-486c-983a-8e26acf4b86d.mp4

## running locally

### requirements

 * x86_64 linux host
 * go >= 1.16
 * [ignite](https://ignite.readthedocs.io/en/stable/installation/)
 * [ipfs cli](https://github.com/ipfs/go-ipfs#install-prebuilt-binaries)
   * NOTE: You must use ipfs v0.11.0 https://ipfs.io/ipns/dist.ipfs.io/go-ipfs/v0.11.0/go-ipfs_v0.11.0_linux-amd64.tar.gz

### pull ignite base image

This prepares the system for running ignite VMs:

```
sudo ignite run binocarlos/bacalhau-ignite-image:v1 \
  --name footwarmer \
  --cpus 1 \
  --memory 1GB \
  --size 10GB \
  --ssh
sudo ignite rm -f footwarmer
```

**NOTE**: for results to be published, each bacalhau node must have a public IP address

You can still run the demo locally, but the results won't be served over the public ipfs network.

### start compute nodes

Have a few terminal windows.

This starts the first compute node listening on port 8080 so we can connect to a known port.

```bash
go run . serve --port 8081 --dev --start-ipfs-dev-only
```

It will also print out the command to run in other terminal windows to connect to this first node (start at least one more so now we have a cluster)

For example:

```bash
go run . serve --peer /ip4/127.0.0.1/tcp/8080/p2p/<peerid> --jsonrpc-port <randomport> --start-ipfs-dev-only
```

### submit a job with the CLI

First we add a data file to run the job against - the `serve` command will have printed out the command to do this - it's just an `ipfs add` command that targets a specific bacalhau node so we can show self selection of jobs working:

```bash
cid=$(IPFS_PATH=data/ipfs/<bacalhau_node_id> ipfs add -q /etc/passwd)
```

Now we submit a job to the network:

```bash
go run . submit --cids=$cid --commands="grep admin /ipfs/$cid"
```

To view the current job status:

```bash
go run . list
go run . list --output json
go run . list --wide
```

### running the demo on seperate servers

When running a real demo (i.e. on different machines with public ips) - here are the things to consider:

 * add the same file to each node using `ipfs add` - so the CID of the job has a file on each node
   * don't use `/etc/passwd` as shown above - it needs to be exactly the same on each node
 * when starting the servers, make sure to use public IP and not `127.0.0.1` in the multi-address used to point at the first node
 * start the ipfs daemon on each node before starting bacalhau
 * don't start the bacalhau daemon with `--dev` or `--start-ipfs-dev-only`

### running the client standalone

If you are running the bacalhau client on a machine that is not running the server - then you must start the ipfs daemon yourself, manually (so we can do `ipfs get` for the results)

## firecracker os image

We use Docker to build the image that firecracker VMs are started with.

The `Dockerfile` lives in `docker/ignite-image/Dockerfile`

To rebuild this image:

```bash
bash scripts/publish-ignite-image.sh
```

NOTE: once you have pushed a new version of the image you must:

```bash
sudo ignite image ls
sudo ignite image rm <id_of_old_image>
```

## local devstack

The easiest way to spin up bacalhau and kick the tires is to use the [devstack](./docs/devstack.md)
