# Bacalhau - The Filecoin Distributed Computation Framework

## demo!

https://user-images.githubusercontent.com/264658/152514573-b7b115ce-4123-486c-983a-8e26acf4b86d.mp4

## running locally

### requirements

 * linux
 * go >= 1.16
 * [ignite](https://ignite.readthedocs.io/en/stable/installation/)
 * [ipfs cli](https://github.com/ipfs/go-ipfs#install-prebuilt-binaries)

### start compute nodes

Have a few terminal windows.

This starts the first compute node listening on port 8080 so we can connect to a known port.

```bash
go run . serve --port 8080 --dev --start-ipfs-dev-only
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


## cli design

```
cid=$(ipfs add -q data.csv)
bac job submit --cids=$cid --commands="sed 's/Office Supplies/Booze/' -i /ipfs/$cid"
```

```
bac job list
```

```
JOB ID      COMMAND              DATA         STATUS
a1b2c3      sed s/Office...      c1d2d3       Submitted
```
```
bac job list



```

* jsonrpc endpoint to list job mempool