---
sidebar_label: 'Quick Start Using Docker'
sidebar_position: 101
---

# Running a Compute Node Using Docker

Good news everyone! You can now run your Bacalhau-IPFS stack in Docker.

This page describes several ways in which to operate Bacalhau. You can choose the method that best suits your needs. The methods are:

* [Connect to and Contribute Resources to the Public Bacalhau Network](#connect-to-the-public-bacalhau-network-using-docker)
* [Run a Private Insecure Local Network for Testing And Development](#run-a-private-bacalhau-network-using-docker-insecure)
* [Run a Private Secure Cluster](#run-a-private-bacalhau-network-using-docker-secure)

### Pre-Prerequisites

* This guide works best on a Linux machine. If you're trying to run this on a Mac, you may encounter issues. Remember that network host mode doesn't work.
* You need to have Docker installed. If you don't have it, you can [install it here](https://docs.docker.com/get-docker/).

## Connect to the Public Bacalhau Network Using Docker

This method is appropriate for those who:

* Provide compute resources to the public Bacalhau network

This is not appropriate for:

* Testing and development
* Running a private network

### Prerequisites

* [Create a new Docker network](#create-a-new-docker-network)

### (Optional) Start a Public IPFS Node

This will start a local IPFS node and connect it to the public DHT. If you already have an IPFS node running, then you can skip this step.

Some notes about this command:

* It wipes the `$(pwd)/ipfs` directory to make sure you have a clean slate
* It runs the IPFS container in the specified Docker network
* It exposes the IPFS API port to the world on port 4002, to avoid clashes with Bacalhau
* It exposes the admin RPC API to the local host only, on port 5001
* We are not specifying or removing the bootstrap nodes, so it will default to connecting to public machines

```bash
# Wipe the current ipfs directory if it exists
rm -rf $(pwd)/ipfs && mkdir $(pwd)/ipfs
# Start the IPFS node
docker run \
    -d --network bacalhau-network --name ipfs_host \
    -v $(pwd)/staging:/export -v $(pwd)/ipfs:/data/ipfs \
    -p 4002:4001 -p 4002:4001/udp \
    -p 127.0.0.1:5001:5001 \
    ipfs/kubo:latest
```

You can now [test that the IPFS node is working](#test-that-the-ipfs-node-is-working).

### Start a Public Bacalhau Node

Bacalhau consists of two parts: a "requester" that is responsible for operating the API and managing jobs, and a "compute" element that is responsible for executing jobs. In a public context, you'd typically just run a compute node, and allow the public requesters to handle the traffic.

Notes about the command:

* It runs the Bacalhau container in "host" mode. This means that the container will use the same network as the host.
* It uses the `root` user, which is the default system user that has access to the Docker socket on a Mac. You may need to change this to suit your environment.
* It mounts the Docker Socket
* It mounts the `/tmp` directory
* It exposes the Bacalhau API ports to the world
* The container version should match that of the current release
* The IPFS connect string points to the RPC port of the IPFS node in Docker. Because Bacalhau is running in the same network, it can use DNS to find the IPFS container IP. If you're running your own node, replace it
* The `--node-type` flag is set to `compute` because we only want to run a compute node
* The `--labels` flag is used to set a human-readable label for the node, and so we can run jobs on our machine later
* We specify the `--peer env` flag so that it uses the environment specified by `BACALHAU_ENVIRONMENT=production` and therefore connects to the public network peers

```bash
sudo docker run \
    -d --rm --name bacalhau \
    --net host \
    --env BACALHAU_ENVIRONMENT=production \
    -u root \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /tmp:/tmp \
    ghcr.io/bacalhau-project/bacalhau:latest \
    serve \
        --ipfs-connect /dns4/localhost/tcp/5001 \
        --node-type compute \
        --labels "owner=docs-quick-start" \
        --private-internal-ipfs=false \
        --peer env
```

There are several ways to ensure that the Bacalhau compute node is connected to the network.

First, check that the Bacalhau libp2p port is open and connected. On Linux you can run `lsof` and it should look something like this:

```bash
❯ sudo lsof -i :1235

COMMAND      PID USER   FD   TYPE    DEVICE SIZE/OFF NODE NAME
bacalhau 1284922 root    3u  IPv4 134301303      0t0  TCP *:1235 (LISTEN)
bacalhau 1284922 root    7u  IPv4 134301307      0t0  UDP *:1235
bacalhau 1284922 root    8u  IPv6 134301308      0t0  TCP *:1235 (LISTEN)
bacalhau 1284922 root    9u  IPv6 134301309      0t0  UDP *:1235
bacalhau 1284922 root   12u  IPv4 134303799      0t0  TCP phil-ethereum-node.europe-west2-c.c.bacalhau-development.internal:1235->191.115.245.35.bc.googleusercontent.com:1235 (ESTABLISHED)
bacalhau 1284922 root   13u  IPv4 134302914      0t0  TCP phil-ethereum-node.europe-west2-c.c.bacalhau-development.internal:1235->251.61.245.35.bc.googleusercontent.com:1235 (ESTABLISHED)
bacalhau 1284922 root   14u  IPv4 134302917      0t0  TCP phil-ethereum-node.europe-west2-c.c.bacalhau-development.internal:1235->239.251.245.35.bc.googleusercontent.com:1235 (ESTABLISHED)
```

Note the three established connections at the bottom. These are the production bootstrap nodes that Bacalhau is now connected to.

You can also check that the node is connected by listing the current network peers and grepping for your IP address or node ID. The node ID can be obtained from the Bacalhau logs. It will look something like this:

```bash
❯ curl -s bootstrap.production.bacalhau.org:1234/peers | jq | grep -A 10 QmaEpsWj4Gw31tBZZ6yagS9ZRSfT8oPgqFuqbvffWJrba5

    "ID": "QmaEpsWj4Gw31tBZZ6yagS9ZRSfT8oPgqFuqbvffWJrba5",
    "Addrs": [
      "/ip4/35.197.229.139/tcp/1235",
      "/ip4/10.154.0.4/tcp/1235",
      "/ip4/127.0.0.1/tcp/1235",
      "/ip4/10.154.0.4/udp/1235/quic",
      "/ip4/127.0.0.1/udp/1235/quic",
      "/ip6/::1/tcp/1235",
      "/ip6/::1/udp/1235/quic"
    ]
  }
```

Finally, submit a job with the label you specified when you ran the compute node. If this label is unique, there should be only one node with this label. The job should succeed. Run the following:

```bash
bacalhau docker run --input=http://example.org/index.html --selector owner=docs-quick-start ghcr.io/bacalhau-project/examples/upload:v1
```

If instead, your job fails with the following error, it means that the compute node is not connected to the network:

```text
Error: failed to submit job: publicapi: after posting request: error starting job: not enough nodes to run job. requested: 1, available: 0
```

## Run a Private Bacalhau Network Using Docker (Insecure)

:::warning
This method is insecure. It does not lock down the IPFS node. Anyone connected to your network can access the IPFS node and read/write data. This is not recommended for production use.
:::

This method is appropriate for:

* Testing and development
* Evaluating the Bacalhau platform before scaling jobs via the public network

This method is useful for testing and development. It's easier to use because it doesn't require a secret IPFS swarm key -- this is essentially an authentication token that allows you to connect to the node.

This method is not appropriate for:

* Secure, private use
* Production use

### Prerequisites

* [Create a new Docker network](#create-a-new-docker-network)

### Start a Local IPFS Node (Insecure)

To run an insecure, private node, you need to initialize your IPFS configuration by removing all of the default public bootstrap nodes. Then we run the node in the normal way, without the special `LIBP2P_FORCE_PNET` flag that checks for a secure private connection.

Some notes about this command:

* It wipes the `$(pwd)/ipfs` directory to make sure you have a clean slate
* It removes the default bootstrap nodes
* It runs the IPFS container in the specified Docker network
* It exposes the IPFS API port to the local host only, to prevent accidentally exposing the IPFS node, on 4002, to avoid clashes with Bacalhau
* It exposes the admin RPC API to the local host only, on port 5001

```bash
# Wipe the current ipfs directory if it exists
rm -rf $(pwd)/ipfs && mkdir $(pwd)/ipfs
# Remove the bootstrap nodes
docker run -t -v $(pwd)/staging:/export -v $(pwd)/ipfs:/data/ipfs ipfs/kubo:latest bootstrap rm --all
# Start the IPFS node
docker run \
    -d --network bacalhau-network --name ipfs_host \
    -v $(pwd)/staging:/export -v $(pwd)/ipfs:/data/ipfs \
    -p 127.0.0.1:4002:4001 -p 127.0.0.1:4002:4001/udp \
    -p 127.0.0.1:8080:8080 -p 127.0.0.1:5001:5001 \
    ipfs/kubo:latest
```

You can now [test that the IPFS node is working](#test-that-the-ipfs-node-is-working).

### Start a Private Bacalhau Node

Bacalhau consists of two parts: a "requester" that is responsible for operating the API and managing jobs, and a "compute" element that is responsible for executing jobs. In a public context, you'd typically just run a compute node, and allow the public requesters to handle the traffic. But in a private context, you'll want to run both.

Notes about the command:

* It runs the Bacalhau container in the specified Docker network
* It uses the `root` user, which is the default system user that has access to the Docker socket on a Mac. You may need to change this to suit your environment
* It mounts the Docker Socket
* It mounts the `/tmp` directory and specifies this as the location where Bacalhau will write temporary execution data (`BACALHAU_NODE_COMPUTESTORAGEPATH`)
* It exposes the Bacalhau API ports to the local host only, to prevent accidentally exposing the API to the public internet
* The container version should match that of the Bacalhau installed on your system
* The IPFS connect string points to the RPC port of the IPFS node. Because Bacalhau is running in the same network, it can use DNS to find the IPFS container IP.
* The `--node-type` flag is set to `requester,compute` because we want to run both a requester and a compute node

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

You can now [test that Bacalhau is working](#test-that-the-bacalhau-node-is-working).

### Run a Job on the Private Network

Now it's time to run a job. Recall that you exposed the Bacalhau API on the default ports to the local host only. So you'll need to use the `--api-host` flag to tell Bacalhau where to find the API. Everything else is a standard part of the Bacalhau CLI.

```bash
bacalhau docker run --api-host=localhost --input=http://example.org/index.html ghcr.io/bacalhau-project/examples/upload:v1
```

The job should succeed. Run it again but this time capture the job ID to make it easier to retrieve the results.

```bash
export JOB_ID=$(bacalhau docker run --api-host=localhost --input=http://example.org/index.html --wait --id-only ghcr.io/bacalhau-project/examples/upload:v1)
```

### Retrieve the Results on the Private Network (Insecure)

To retrieve the results using the Bacalhau CLI, you need to know the p2p swarm multiaddress of the IPFS node because you don't want to connect to the public global IPFS network. To do that you can run the IPFS id command (and parse to remove the trub at the bottom of the barrel):

```bash
export SWARM_ID=$(docker run -t --rm --network=bacalhau-network ipfs/kubo:latest --api=/dns4/ipfs_host/tcp/5001 id -f="<id>" | tail -n 1)
export SWARM_ADDR=/ip4/127.0.0.1/tcp/4002/p2p/$SWARM_ID
```

Note that the command above changes the reported port from 4001 to 4002. This is because the IPFS node is running on port 4002, but the IPFS id command reports the port as 4001.

Now get the results:

```bash
rm -rf results && mkdir results && \
    bacalhau --api-host=localhost --ipfs-swarm-addrs=$SWARM_ADDR get --output-dir=results $JOB_ID
```

Alternatively, you can use the Docker container, mount the results volume, and change the `--api-host` to the name of the Bacalhau container and the `--ipfs-swarm-addrs` back to port 4001:

```bash
rm -rf results && mkdir results && \
docker run -t --rm --network=bacalhau-network \
    -v $(pwd)/results:/results \
    ghcr.io/bacalhau-project/bacalhau:latest \
    get --api-host=bacalhau --ipfs-swarm-addrs=/dns4/bacalhau/tcp/4001/p2p/$SWARM_ID --output-dir=/results $JOB_ID
```

## Run a Private Bacalhau Network Using Docker (Secure)

Running a private secure network is useful in a range of scenarios, including:

* Running a private network for a private project

You need two things. A private IPFS node to store data and a Bacalhau node to execute over that data. To keep the nodes private you need to tell the nodes to shush and use a secret key. This is a bit harder to use, and a bit more involved than the insecure version.

### Prerequisites

* [Create a new Docker network](#create-a-new-docker-network)

### Start a Private IPFS Node (Secure)

:::warning
Private IPFS nodes are experimental. [See the IPFS documentation](https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md#private-networks) for more information.
:::

First, you need to bootstrap a new IPFS cluster for your own private use. This consists of a process of generating a swarm key, removing any bootstrap nodes, and then starting the IPFS node.

Some notes about this command:

* It wipes the `$(pwd)/ipfs` directory to make sure you have a clean slate
* It generates a new swarm key -- this is the token that is required to connect to this node
* It removes the default bootstrap nodes
* It runs the IPFS container in the specified Docker network
* It exposes the IPFS API port to the local host only, to prevent accidentally exposing the IPFS node, on 4002, to avoid clashes with Bacalhau
* It exposes the admin RPC API to the local host only, on port 5001

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

### Start a Private Bacalhau Node (Secure)

The instructions to run a secure private Bacalhau network are the same as the insecure version, [please follow those instructions](#start-a-private-bacalhau-node-secure).

### Run a Job on the Private Network (Secure)

The instructions to run a job are the same as the insecure version, [please follow those instructions](#run-a-job-on-the-private-network).

### Retrieve the Results on the Private Network (Secure)

The same process as above can be used to retrieve results from the IPFS node as long as the Bacalhau `get` command
has access to the IPFS swarm key.

Running the Bacalhau binary from outside of Docker:

```bash
bacalhau --api-host=localhost --ipfs-swarm-addrs=$SWARM_ADDR --ipfs-swarm-key=$(pwd)/ipfs/swarm.key get $JOB_ID
```

Alternatively, you can use the Docker container, mount the results volume, and change the `--api-host` to the name of the Bacalhau container and the `--ipfs-swarm-addrs` back to port 4001:

```bash
mkdir results && \
docker run -t --rm --network=bacalhau-network \
    -v $(pwd)/results:/results \
    -v $(pwd)/ipfs:/ipfs \
    ghcr.io/bacalhau-project/bacalhau:latest \
    get --api-host=bacalhau --ipfs-swarm-addrs=/dns4/bacalhau/tcp/4001/p2p/$SWARM_ID --ipfs-swarm-key=/ipfs/swarm.key --output-dir=/results $JOB_ID
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
|DOCKER_PASSWORD|A read-only access token, generated from the page at [https://hub.docker.com/settings/security](https://hub.docker.com/settings/security)> |

:::info
Currently, this authentication is only available (and required) by the [Docker Hub](https://hub.docker.com/)
:::
