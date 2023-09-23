---
sidebar_label: 'Storage Providers'
sidebar_position: 130
---

# Storage Providers

Bacalhau has two ways to make use of external storage providers:

 * **Inputs** storage resources consumed as inputs to jobs
 * **Publishers** storage resources created with the results of jobs

## Inputs

### IPFS

To start, you'll need to connect the Bacalhau node to an IPFS server so that you can run jobs that consume CIDs as inputs.

You can either [install IPFS](https://docs.ipfs.tech/install/) and run it locally, or you can connect to a remote IPFS server.

In both cases, you should have an [IPFS multiaddress](https://richardschneider.github.io/net-ipfs-core/articles/multiaddress.html) for the IPFS server that should look something like this:

```bash
export IPFS_CONNECT=/ip4/10.1.10.10/tcp/80/p2p/QmVcSqVEsvm5RR9mBLjwpb2XjFVn5bPdPL69mL8PH45pPC
```

:::caution

The multiaddress above is just an example - you'll need to get the multiaddress of the IPFS server you want to connect to.

:::

You can then configure your Bacalhau node to use this IPFS server by passing the `--ipfs-connect` argument to the `serve` command:

```bash
bacalhau serve --ipfs-connect $IPFS_CONNECT
```

Or, set the `Node.IPFS.Connect` property in the Bacalhau configuration file.

## Publishers

### IPFS

The IPFS publisher works using the same setup as above - you'll need to have an
IPFS server running and a multiaddress for it. You'll then you pass that
multiaddress using the `--ipfs-connect` argument to the `serve` command.

If you are publishing to a public IPFS node, you can use `bacalhau get` with no
further arguments to download the results. However, you may experience a delay
in results becoming available as indexing of new data by public nodes takes
time.

To speed up the download or to retrieve results from a private IPFS node, pass
the swarm multiaddress to `bacalhau get` to download results.

```bash
# Set the below environment variable, use the --ipfs-swarm-addrs flag,
# or set the Node.IPFS.SwarmAddresses config property.
export BACALHAU_IPFS_SWARM_ADDRESSES=/ip4/.../tcp/5001/p2p/Qmy...
bacalhau get $JOB_ID
```

Pass the swarm key to `bacalhau get` if the IPFS swarm is a private swarm.

```bash
# Set the below environment variable, use the --ipfs-swarm-key flag,
# or set the Node.IPFS.SwarmKeyPath config property.
export BACALHAU_IPFS_SWARM_KEY=./path/to/swarm.key
bacalhau get $JOB_ID
```
