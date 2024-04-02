---
sidebar_label: Running Locally
---

# Running locally with the 'devstack' command

You can run a stand-alone Bacalhau and IPFS network on your computer with the
following guide.

The `devstack` command of `bacalhau` will start a 3 node cluster alongside
isolated ipfs servers.

This is useful to kick the tires and/or developing on the codebase. It's also
the tool used by some tests.

## Pre-requisites

- x86_64 or ARM64 architecture
  - Ubuntu 20.0+ has most often been used for development and testing
- Go >= 1.21
- (Optional) [Docker Engine](https://docs.docker.com/get-docker/)
- (Optional) A build of the [latest Bacalhau
  release](https://github.com/bacalhau-project/bacalhau/releases/)

## (Optional) Building Bacalhau from source

```bash
sudo apt-get update && sudo apt-get install -y make gcc zip nodejs npm
sudo snap install go --classic
wget https://github.com/bacalhau-project/bacalhau/archive/refs/heads/main.zip
unzip main.zip
cd bacalhau-main
make build
```

## Start the cluster

```bash
./bacalhau devstack
```

This will start a 3 node Bacalhau cluster connected with libp2p.

Each node has its own IPFS server isolated using the `IPFS_PATH` environment
variable and its own API RPC server isolated using a random port. These IPFS
nodes are not connected to the public IPFS network. If you wish to connect the
devstack to the public IPFS network, you can include `--public-ipfs` flag.

You can also use your own IPFS node and connect it to the devstack by running
(after starting the devstack):

```
# $BACALHAU_IPFS_SWARM_ADDRESSES is an environment variable given after starting devstack
ipfs swarm connect $BACALHAU_IPFS_SWARM_ADDRESSES
```

If you would like to make it a bit more predictable and/or ignore errors (such
as during CI), you can add the following before your execution:

```
IGNORE_PID_AND_PORT_FILES=true PREDICTABLE_API_PORT=1
```

Once everything has started up - you will see output like the following:

```bash

Devstack is ready!
To use the devstack, run the following commands in your shell:
export BACALHAU_IPFS_SWARM_ADDRESSES=/ip4/127.0.0.1/tcp/33033/p2p/QmNp5XqbkePNYtRzB2MXZPo6MxkeH6N2fYZRCLT57VsACn
export BACALHAU_API_HOST=0.0.0.0
export BACALHAU_API_PORT=39763

By default devstack is not running on public IPFS network.
If you wish to connect devstack to public IPFS network consider running new IPFS node daemon locally
and then connecting it to bacalhau using the command below or by adding --public-ipfs flag:

ipfs swarm connect $BACALHAU_IPFS_SWARM_ADDRESSES
```

The message above contains the environment variables you need for a new window.
You can paste these into a new terminal so that bacalhau will use your local
devstack.

Alternatively, to remove the need to copy and paste, you can set
`DEVSTACK_ENV_FILE` environment variable to the name of a .env file that
devstack will write to, and bacalhau commands will read from, before launching
the devstack e.g.:

```bash
DEVSTACK_ENV_FILE=.devstack.env bacalhau devstack
```

When the devstack is shut down, the local env file (if configured) will be
removed. It is suggested you use `.devstack.env` to avoid clashing with longer
lived `.env` files.

## New Terminal Window

- Open an additional terminal window to be used for submitting jobs.
- Copy and paste environment variables from previous message into this window.
  EG:

```bash
export BACALHAU_IPFS_SWARM_ADDRESSES=/ip4/127.0.0.1/tcp/33033/p2p/QmNp5XqbkePNYtRzB2MXZPo6MxkeH6N2fYZRCLT57VsACn
export BACALHAU_API_HOST=0.0.0.0
export BACALHAU_API_PORT=62406
```

You are now ready to submit a job to your local devstack.

## Submit a simple job

This will submit a simple job to a single node:

```bash
./bacalhau docker run ubuntu echo "hello devstack test"
```

This should output something like the following:

```bash
15:01:00.638 | INF bacalhau/utils.go:69 > Development client version, skipping version check
d7d4d23d-08ff-46f4-a695-f37647da67cc
```

After a short while - the job should be in `complete` state.

```bash
./bacalhau list --wide
 CREATION_TIME      ID                                    JOB                             STATE      RESULT
 22-08-29-15:01:00  d7d4d23d-08ff-46f4-a695-f37647da67cc  Docker ubuntu echo hello world  Published  /ipfs/QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf
```

Download the results to the current directory:

```bash
./bacalhau get d7d4d23d-08ff-46f4-a695-f37647da67cc # Works with partial IDs - just the first 8 characters
```

You should now have the following files and directories:

- stdout
- stderr
- volumes/output

If you `cat stdout` it should read "hello devstack test". If you write any files
in your job, they will appear in volumes/output.
