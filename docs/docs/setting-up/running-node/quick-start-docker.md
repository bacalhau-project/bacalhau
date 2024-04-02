---
sidebar_label: 'Quick Start Using Docker'
sidebar_position: 101
---

# Quick Start Using Docker

This page describes a way in which to operate Bacalhau-IPFS stack in Docker.

### Pre-Prerequisites

1. This guide works best on a Linux machine. If you're trying to run this on a Mac, you may encounter issues. Remember that network host mode doesn't work.
1. You need to have Docker installed. If you don't have it, you can [install it here](https://docs.docker.com/get-docker/).


## Run a Private Bacalhau Network Using Docker

Running a private secure network is useful in a range of scenarios, including testing, development and utilization of available resources for a private project.

You need two things:
1. A private IPFS node to store data
1. A Bacalhau node to execute over that data. 

To keep the nodes private you need to tell the nodes to shush and use a secret key.

### Prerequisite

Before you start, you need to [сreate a new Docker network](./quick-start-docker.md#create-a-new-docker-network).

### Start a Private IPFS Node

:::warning
Private IPFS nodes are experimental. See the [IPFS documentation](https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md#private-networks) for more information.
:::

First, you need to bootstrap a new IPFS cluster for your own private use. This consists of a process of generating a swarm key, removing any bootstrap nodes, and then starting the IPFS node.

Notes about this command:

1. It wipes the `$(pwd)/ipfs` directory to make sure you have a clean slate
1. It generates a new swarm key - this is the token that is required to connect to this node
1. It removes the default bootstrap nodes
1. It runs the IPFS container in the specified Docker network
1. It exposes the IPFS API port to the local host only, to prevent accidentally exposing the IPFS node, on 4002, to avoid clashes with Bacalhau
1. It exposes the admin RPC API to the local host only, on port 5001

```bash
# Wipe the current ipfs directory if it exists
rm -rf $(pwd)/ipfs && mkdir $(pwd)/ipfs

# Create a new swarm key -- a secret key that will be used to bootstrap the private network
echo -e "/key/swarm/psk/1.0.0/\n/base16/\n`tr -dc 'a-f0-9' < /dev/urandom | head -c64`" > $(pwd)/ipfs/swarm.key

# Remove the bootstrap nodes
docker run -t -v $(pwd)/staging:/export -v $(pwd)/ipfs:/data/ipfs ipfs/kubo:latest bootstrap rm --all

# Start the IPFS node
docker run \
    -d --network bacalhau-network --name ipfs_host \
    -e LIBP2P_FORCE_PNET=1 \
    -v $(pwd)/staging:/export -v $(pwd)/ipfs:/data/ipfs \
    -p 127.0.0.1:4002:4001 -p 127.0.0.1:4002:4001/udp \
    -p 127.0.0.1:8080:8080 -p 127.0.0.1:5001:5001 \
    ipfs/kubo:latest
```

### Start a Private Bacalhau Node

Bacalhau consists of two types of nodes: a `requester` node, that is responsible for operating the API and managing jobs, and a `compute` node, that is responsible for executing jobs. In a private context you'll have to run both.

Notes about the command:

1. It runs the Bacalhau container in the specified Docker network
1. It uses the `root` user, which is the default system user that has access to the Docker socket on a Mac. You may need to change this to suit your environment
1. It mounts the Docker Socket
1. It mounts the `/tmp` directory and specifies this as the location where Bacalhau will write temporary execution data (`BACALHAU_NODE_COMPUTESTORAGEPATH`)
1. It exposes the Bacalhau API ports to the local host only, to prevent accidentally exposing the API to the public internet
1. The container version should match that of the Bacalhau installed on your system
1. The IPFS connect string points to the RPC port of the IPFS node. Because Bacalhau is running in the same network, it can use DNS to find the IPFS container IP
1. The `--node-type` flag is set to `requester,compute` because we want to run both a requester and a compute node

```bash
docker run \
    -d --network bacalhau-network --name bacalhau \
    -u root \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /tmp:/tmp \
    -e BACALHAU_NODE_COMPUTESTORAGEPATH=/tmp \
    -p 127.0.0.1:1234:1234 -p 127.0.0.1:1235:1235 -p 127.0.0.1:1235:1235/udp \
    ghcr.io/bacalhau-project/bacalhau:latest \
    serve \
        --ipfs-connect /dns4/ipfs_host/tcp/5001 \
        --node-type requester,compute
```

You can now [test that Bacalhau is working](./quick-start-docker.md#test-that-the-bacalhau-node-is-working).

### Run a Job on the Private Network

Now it's time to run a job. Recall that you exposed the Bacalhau API on the default ports to the local host only. So you'll need to use the `--api-host` flag to tell Bacalhau where to find the API. Everything else is a standard part of the Bacalhau CLI.

```bash
bacalhau docker run \
    --api-host=localhost \
    --input=http://example.org/index.html \
    ghcr.io/bacalhau-project/examples/upload:v1
```

The job should succeed. Run it again but this time capture the job ID to make it easier to retrieve the results.

```bash
export JOB_ID=$(bacalhau docker run \
    --api-host=localhost \
    --input=http://example.org/index.html \
    --wait \
    --id-only ghcr.io/bacalhau-project/examples/upload:v1)
```

### Retrieve the Results on the Private Network

The same process as above can be used to retrieve results from the IPFS node as long as the Bacalhau `get` command has access to the IPFS swarm key.

Running the Bacalhau binary from outside of Docker:

```bash
bacalhau get $JOB_ID \
--api-host=localhost \
--ipfs-swarm-addrs=$SWARM_ADDR \
--ipfs-swarm-key=$(pwd)/ipfs/swarm.key 
```

Alternatively, you can use the Docker container, mount the results volume, and change the `--api-host` to the name of the Bacalhau container and the `--ipfs-swarm-addrs` back to port 4001:

```bash
mkdir results && \
docker run -t --rm --network=bacalhau-network \
    -v $(pwd)/results:/results \
    -v $(pwd)/ipfs:/ipfs \
    ghcr.io/bacalhau-project/bacalhau:latest \
    get $JOB_ID \
    --api-host=bacalhau \
    --ipfs-swarm-addrs=/dns4/bacalhau/tcp/4001/p2p/$SWARM_ID \
    --ipfs-swarm-key=/ipfs/swarm.key \
    --output-dir=/results 
```


## Common Prerequisites

### Create a New Docker Network

Without this, inter-container DNS will not work, and internet access may not work either.

```bash
docker network create --driver bridge bacalhau-network
```

:::tip

Double check that this network can access the internet (so Bacalhau can call external URLs).

```bash
docker run --rm --network bacalhau-network alpine ping -c 2 bacalhau.org
```

This should be successful. If it is not, then please troubleshoot your docker networking. For example, on my Mac, I had to totally uninstall Docker, restart the computer, and then reinstall Docker. Then it worked. Also check https://docs.docker.com/desktop/troubleshoot/known-issues/. Apparently "ping from inside a container to the Internet does not work as expected.". No idea what that means. How do you break ping?

:::

### Test that the IPFS Node is Working

You can now browse the IPFS web UI at http://127.0.0.1:5001/webui.

Read more about the IPFS docker image [here](https://docs.ipfs.tech/install/run-ipfs-inside-docker/#set-up).

:::warning
As described in [their documentation](https://docs.ipfs.tech/reference/kubo/rpc/#getting-started), never expose the RPC API port (port 5001) to the public internet.
:::

### Test that the Bacalhau Node is Working

Ensure that the Bacalhau logs (`docker logs bacalhau`) have no errors.

Check that your Bacalhau installation is the same version:

```bash
bacalhau --api-host=localhost version
```

The versions should match. Alternatively, you can use the Docker container:

```bash
docker run --network=bacalhau-network --rm ghcr.io/bacalhau-project/bacalhau:latest --api-host=bacalhau version
```

Perform a list command to ensure you can connect to the Bacalhau API.

```bash
bacalhau --api-host=localhost list
```

It should return empty.

### Authenticate with docker hub

If you are retrieving and running images from [docker hub](https://hub.docker.com/) you
may encounter issues with rate-limiting. Docker provides higher limits when authenticated, the size of the limit is based on the type of your account.

Should you wish to authenticate with Docker Hub when pulling images, you can do so
by specifying credentials as environment variables wherever your compute node is running.

|Environment variable|Description|
|---|---|
|DOCKER_USERNAME|The username with which you are registered at [https://hub.docker.com/](https://hub.docker.com/)|
|DOCKER_PASSWORD|A read-only access token, generated from the page at [https://hub.docker.com/settings/security](https://hub.docker.com/settings/security)|

:::info
Currently, this authentication is only available (and required) by the [Docker Hub](https://hub.docker.com/)
:::
