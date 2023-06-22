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

You can then configure your bacalhau node to use this IPFS server by passing the `--ipfs-connect` argument to the `serve` command:

```bash
bacalhau serve \
  --ipfs-connect $IPFS_CONNECT
```

## Publishers

### Estuary

[Estuary](https://estuary.tech/) gives an accessible API for adding content to both IPFS and Filecoin.

You can configure your bacalhau node to use Estuary by passing the `--estuary-api-key` argument to the `serve` command:

```bash
bacalhau serve \
  --estuary-api-key XXX
```

To get an API key for estuary, you'll need to [register an account](https://estuary.tech/) and then [create an api key](https://estuary.tech/api-admin).

### IPFS

The IPFS publisher works using the same setup as above - you'll need to have an IPFS server running, a multiaddress for it. You'll then you pass that multiaddress using the `--ipfs-connect` argument to the `serve` command.

The IPFS publisher will be used as the default if there is no Estuary API Key configured.
