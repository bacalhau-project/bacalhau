---
sidebar_label: 'Quick Start'
sidebar_position: 100
toc_max_heading_level: 4
---

# Join as Compute Provider

Bacalhau is a peer-to-peer network of compute providers that will run jobs submitted by users. A Compute Provider (CP) is anyone who is running a Bacalhau compute node participating in the Bacalhau compute network, regardless of whether they are hosting any Filecoin data.

This section will show you how to configure and run a bacalhau node and start accepting and running jobs.

To bootstrap your node and join the network as a CP you can leap right into the [Ubuntu 22.04 quick start](#quick-start-ubuntu-2204) below, or find for more setup details in these guides:

* [Networking](https://docs.bacalhau.org/running-node/networking)
* [Storage Providers](https://docs.bacalhau.org/running-node/storage-providers)
* [Job Selection Policy](https://docs.bacalhau.org/running-node/job-selection)
* [Resource Limits](https://docs.bacalhau.org/running-node/resource-limits)
* [GPU Support](https://docs.bacalhau.org/running-node/gpu)
* [Windows Support](https://docs.bacalhau.org/running-node/windows-support) (with limitations)

:::info

If you run on a different system than Ubuntu, drop us a message on [Slack](https://join.slack.com/t/bacalhauproject/shared_invite/zt-1sihp4vxf-TjkbXz6JRQpg2AhetPzYYQ/archives/C02RLM3JHUY)!
We'll add instructions for your favorite OS.

:::

## Quick start (Ubuntu 22.04)

Estimated time for completion: 10 min.

Tested on: Ubuntu 22.04 LTS (x86/64) running on a GCP e2-standard-4 (4 vCPU, 16 GB memory) instance with 40 GB disk size.

### Prerequisites

* Docker Engine - to take on Docker workloads
* Connection to storage provider - for storing job's results
* Firewall - to ensure your node can communicate with the rest of the network
* Physical hardware, Virtual Machine or cloud based host. A Bacalhau compute node is not intended to be ran from within a Docker container.

#### Install Docker

To run docker based workloads, you should have docker installed and running.

If you already have it installed and want to configure the connection to Docker with the following environment variables:

* `DOCKER_HOST` to set the url to the docker server.
* `DOCKER_API_VERSION` to set the version of the API to reach, leave empty for latest.
* `DOCKER_CERT_PATH` to load the TLS certificates from.
* `DOCKER_TLS_VERIFY` to enable or disable TLS verification, off by default.

If you do not have Docker on your system, you can follow the official [docker installation instructions](https://docs.docker.com/engine/install/) or just use the snippet below:

```bash
# install dependencies
sudo apt-get update
sudo apt-get install \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

# add package repo and key
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# install docker
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin
```

Now make [Docker manageable by a non-root user](https://docs.docker.com/engine/install/linux-postinstall/#:~:text=The%20Docker%20daemon%20always%20runs,members%20of%20the%20docker%20group):

```bash
sudo groupadd docker
sudo usermod -aG docker $USER
```

#### Ensure your Storage Server is Running

We will need to connect our bacalhau node to an storage server for this we will be using IPFS server so we can run jobs that consume CIDs as inputs.

You can either install and run it locally or you can connect to a remote IPFS server.

In both cases - we should have a [multiaddress](https://richardschneider.github.io/net-ipfs-core/articles/multiaddress.html) for our IPFS server that should look something like this:

```bash
export IPFS_CONNECT=/ip4/10.1.10.10/tcp/80/p2p/QmVcSqVEsvm5RR9mBLjwpb2XjFVn5bPdPL69mL8PH45pPC
```

:::caution

The multiaddress above is just an example - you need to get the multiaddress of the server you want to connect to.

:::

To install a single IPFS node locally on Ubuntu you can follow the [official instructions](https://docs.ipfs.tech/install/ipfs-desktop/#ubuntu), or follow the steps below. We advise to run the same IPFS version as the Bacalhau main network.

Using the command below:

```bash
export IPFS_VERSION=$(wget -q -O - https://raw.githubusercontent.com/filecoin-project/bacalhau/main/ops/terraform/production.tfvars | grep --color=never ipfs_version | awk -F'"' '{print $2}')
```

Install:

```bash
wget "https://dist.ipfs.tech/go-ipfs/${IPFS_VERSION}/go-ipfs_${IPFS_VERSION}_linux-amd64.tar.gz"
tar -xvzf "go-ipfs_${IPFS_VERSION}_linux-amd64.tar.gz"
cd go-ipfs
sudo bash install.sh
ipfs --version
```

Configure:

```bash
sudo mkdir -p /data/ipfs
export IPFS_PATH=/data/ipfs
sudo chown $(id -un):$(id -gn) ${IPFS_PATH} # change ownership of ipfs directory
ipfs init
```

Now launch the IPFS daemon **in a separate terminal** (make sure to export the `IPFS_PATH` environment variable there as well):

```bash
ipfs daemon
```

:::info

If you want to run the IPFS daemon as a [systemd](https://en.wikipedia.org/wiki/Systemd) service, here's an example [systemd service file](https://github.com/bacalhau-project/bacalhau/blob/main/ops/terraform/remote_files/configs/ipfs.service).

:::

Don't forget we need to fetch an [IPFS multiaddress](https://richardschneider.github.io/net-ipfs-core/articles/multiaddress.html) pointing to our local node.

Use the following command to print out a number of addresses.

```bash
ipfs id
```

```json
{
        "ID": "12D3KooWNhRU6H1eeqvT1jQAXFAdFDvT5H4AEGmHV7N1cYbzYh1F",
        "PublicKey": "CAESIL9gmDyR6IgM7ym1JmJ8JKL7NvEgIEGaWwssanl1ieuW",
        "Addresses": [
                "/ip4/10.128.0.11/tcp/4001/p2p/12D3KooWNhRU6H1eeqvT1jQAXFAdFDvT5H4AEGmHV7N1cYbzYh1F",
                "/ip4/10.128.0.11/udp/4001/quic/p2p/12D3KooWNhRU6H1eeqvT1jQAXFAdFDvT5H4AEGmHV7N1cYbzYh1F",
                "/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWNhRU6H1eeqvT1jQAXFAdFDvT5H4AEGmHV7N1cYbzYh1F",
                "/ip4/127.0.0.1/udp/4001/quic/p2p/12D3KooWNhRU6H1eeqvT1jQAXFAdFDvT5H4AEGmHV7N1cYbzYh1F",
                ...
        ],
        "AgentVersion": "go-ipfs/0.12.2/",
        "ProtocolVersion": "ipfs/0.1.0",
        "Protocols": [
                "/ipfs/bitswap",
                ...
                "/p2p/id/delta/1.0.0",
                "/x/"
        ]
}
```

I pick the record that combines `127.0.0.1` and `tcp` but I replace port `4001` with `5001`:

```bash
export IPFS_CONNECT=/ip4/127.0.0.1/tcp/5001/p2p/12D3KooWNhRU6H1eeqvT1jQAXFAdFDvT5H4AEGmHV7N1cYbzYh1F
```

#### Configure firewall

To ensure that our node can communicate with other nodes on the network - we need to make sure the **1235** port is open.

(Optional) To ensure the cli can communicate with our node directly (i.e. `bacalhau --api-host <MY_NODE_PUBLIC_IP> version`) - we need to make sure the **1234** port is open.

Firewall configuration is very specific to your network and we can't provide generic instructions for this step but if you need any help feel free to reach out on [Slack!](https://join.slack.com/t/bacalhauproject/shared_invite/zt-1sihp4vxf-TjkbXz6JRQpg2AhetPzYYQ/archives/C02RLM3JHUY)

### Install the Bacalhau Binary

[Install the bacalhau binary](/getting-started/installation#install-the-bacalhau-client) to run `bacalhau serve`.

:::info

If you want to run Bacalhau  as a [systemd](https://en.wikipedia.org/wiki/Systemd) service, here's an example [systemd service file](https://github.com/bacalhau-project/bacalhau/blob/main/ops/terraform/remote_files/configs/bacalhau.service).

:::

### Run bacalhau

Now we can run our bacalhau node:

```bash
LOG_LEVEL=debug BACALHAU_ENVIRONMENT=production \
  bacalhau serve \
    --node-type compute \
    --ipfs-connect $IPFS_CONNECT \
    --private-internal-ipfs=false \
    --peer env
```

Alternatively, you can run the following Docker command:

```bash
docker run -it --rm \
  -e LOG_LEVEL=debug \
  -e BACALHAU_ENVIRONMENT=production \
  ghcr.io/bacalhau-project/bacalhau:latest serve \
    --node-type compute \
    --ipfs-connect $IPFS_CONNECT \
    --private-internal-ipfs=false \
    --peer env
```

These commands join this node to the public Bacalhau network, congrats! :tada:

### Check your node works

Even though the cli (by default) submits jobs, each node listens for events on the global network and possibly bids for taking a job: your logs should therefore show activity of your node bidding for incoming jobs.

To quick check your node runs properly, let's submit the following dummy job:

```bash
bacalhau docker run ubuntu echo Test
```

If you see logs of your computenode bidding for the job above it means you've successfully joined Bacalhau as a Compute Provider!

### What's next?

At this point you probably have a number of questions for us. What incentive should you expect for running a public Bacalhau node?
Please contact us on [Slack](https://join.slack.com/t/bacalhauproject/shared_invite/zt-1sihp4vxf-TjkbXz6JRQpg2AhetPzYYQ/archives/C02RLM3JHUY) to further discuss this topic and for sharing your valuable feedback.
