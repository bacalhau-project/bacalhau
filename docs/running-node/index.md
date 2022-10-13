---
sidebar_label: 'Overview'
sidebar_position: 100
---

# Join as Compute Provider

Bacalhau is a peer-to-peer network of compute providers that will run jobs submitted by users. A Compute Provider (CP) is anyone who is running a Bacalhau compute node participating in the Bacalhau compute network, regardless of whether they are hosting any Filecoin data.

This section will show you how to configure and run a bacalhau node and start accepting and running jobs.

To bootstrap your node and join the network as a CP you can leap right into the [Ubuntu 22.04 quick start](#quick-start-ubuntu-2204) below, or find for more setup details in these guides:

  * [Networking](networking)
  * [Storage Providers](storage-providers)
  * [Job Selection Policy](job-selection)
  * [Resource Limits](resource-limits)
  * [GPU Support](gpu)
  * [Windows Support](windows-support)

:::info

If you run on a different system than Ubuntu, drop us a message on [Slack](https://filecoinproject.slack.com/archives/C02RLM3JHUY)! 
We'll add instructions for your favourite OS.

:::
## Quick start (Ubuntu 22.04)


Tested on: Ubuntu, 22.04 LTS (x86/64)
### Install the bacalhau binary
First, you should [install the bacalhau binary](/getting-started/installation#prerequisite-install-bacalhau-client) to run `bacalhau serve`. 
### Install docker
To run docker based workloads, you should have docker installed and running.

If you already have it installed and want to configure the connection to Docker with the following environment variables:

 * `DOCKER_HOST` to set the url to the docker server.
 * `DOCKER_API_VERSION` to set the version of the API to reach, leave empty for latest.
 * `DOCKER_CERT_PATH` to load the TLS certificates from.
 * `DOCKER_TLS_VERIFY` to enable or disable TLS verification, off by default.

If you do not have Docker on your system, you can follow the official [docker installation instructions](https://docs.docker.com/engine/install/) or just use the snippet below:

```
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
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# install docker
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin
```


### Ensure IPFS is running
We will need to connect our bacalhau node to an IPFS server so we can run jobs that consume CIDs as inputs.


You can either install IPFS and run it locally or you can connect to a remote IPFS server.

In both cases - we should have an [IPFS multiaddress](https://richardschneider.github.io/net-ipfs-core/articles/multiaddress.html) for our IPFS server that should look something like this:

```bash
export IPFS_CONNECT=/ip4/10.1.10.10/tcp/80/p2p/QmVcSqVEsvm5RR9mBLjwpb2XjFVn5bPdPL69mL8PH45pPC
```

:::caution

The multiaddress above is just an example - you need to get the multiaddress of the IPFS server you want to connect to.

:::

To install a single IPFS node locally on Ubuntu you can follow the [official instructions](https://docs.ipfs.tech/install/ipfs-desktop/#ubuntu), or follow the steps below.

We advise to run the same IPFS version as the Bacalhau main network.
Pick that with the command below:

```
export IPFS_VERSION=$(wget -q -O - https://raw.githubusercontent.com/filecoin-project/bacalhau/main/ops/terraform/production.tfvars | grep --color=never ipfs_version | awk -F'"' '{print $2}')
```

Install:

```
wget "https://dist.ipfs.tech/go-ipfs/${IPFS_VERSION}/go-ipfs_${IPFS_VERSION}_linux-amd64.tar.gz"
tar -xvzf "go-ipfs_${IPFS_VERSION}_linux-amd64.tar.gz"
cd go-ipfs
sudo bash install.sh
ipfs --version
```

Configure:

```
sudo mkdir -p /data/ipfs
export IPFS_PATH=/data/ipfs
sudo chown <user>:<group> ${IPFS_PATH}
ipfs init
```

Now launch the IPFS daemon in a separate terminal (make sure to export the `IPFS_PATH` environment variable there as well):

```
ipfs daemon
```

If you want to run the IPFS daemon as a [systemd](https://en.wikipedia.org/wiki/Systemd) feel free to reuse [this service configuration file](https://github.com/filecoin-project/bacalhau/blob/main/ops/terraform/remote_files/configs/ipfs-daemon.service). 

Don't forget we need to fetch a [IPFS multiaddress](https://richardschneider.github.io/net-ipfs-core/articles/multiaddress.html) of our local IPFS node.
The following command prints out a number of addresses.

<!-- TODO  need port 5001 instead of 4001 - clarify -->

```bash
ipfs id
export IPFS_CONNECT=/ip4/127.0.0.1/tcp/5001/p2p/12D3KooWDkArGGx55V3eg65qW8dxyPWX5ed7XWJCiqBBnABLRsw8
```

### Configure firewall

To ensure that our node can communicate with other nodes on the network - we need to make sure the **1235** port is open.

Firewall configuration is very specific to your network and we can't provide generic instructions for this step but if you need any help feel free to reach out on [Slack!](https://filecoinproject.slack.com/archives/C02RLM3JHUY)

### Run bacalhau

Now we can run our bacalhau node:

<!-- TODO  sudo should not be needed - clarify -->

```bash
LOG_LEVEL=debug bacalhau serve \
  --ipfs-connect $IPFS_CONNECT
```