---
sidebar_label: 'Overview'
sidebar_position: 100
---

# Overview

Bacalhau is a peer-to-peer network of compute providers that will run jobs submitted by users.

This section will show you how to configure and run a bacalhau node and start accepting and running jobs.

You can leap right into the [quick start](#quickstart) or you can follow these guides:

 * [Install](install)
 * [Networking](networking)
 * [Storage Providers](storage-providers)
 * [Job Selection](job-selection)
 * [Resource Limits](resource-limits)
 * [GPU Support](gpu)

## Quick start

### Install bacalhau binary
So we can run `bacalhau serve` - we first need to [install the bacalhau binary](/getting-started/installation#prerequisite-install-bacalhau-client)

### Install docker
So we can run docker based workloads - we need to have [docker installed](https://docs.docker.com/engine/install/) and running.

### Ensure IPFS is running
We will need to connect our bacalhau node to an IPFS server so we can run jobs that consume CIDs as inputs.

You can either [install IPFS](https://docs.ipfs.tech/install/) and run it locally or you can connect to a remote IPFS server.

In both cases - we should have an [IPFS multiaddress](https://richardschneider.github.io/net-ipfs-core/articles/multiaddress.html) for our IPFS server that should look something like this:

```bash
export IPFS_CONNECT=/ip4/10.1.10.10/tcp/80/p2p/QmVcSqVEsvm5RR9mBLjwpb2XjFVn5bPdPL69mL8PH45pPC
```

:::caution

The multiaddress above is just an example - you need to get the multiaddress of the IPFS server you want to connect to.

:::

### Configure firewall

To ensure that our node can communicate with other nodes on the network - we need to make sure the **1235** port is open.

### Run bacalhau

Now we can run our bacalhau node:

```bash
bacalhau serve \
  --ipfs-connect $IPFS_CONNECT
```