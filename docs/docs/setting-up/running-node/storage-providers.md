---
sidebar_label: 'Storage Providers'
sidebar_position: 130
---

# Storage Providers

Bacalhau has two ways to make use of external storage providers:

1. **Inputs**: storage resource from where data is loaded to be processed in jobs
2. **Publishers**: storage resource where the results of job execution are uploaded

## Inputs

### IPFS

To start, you'll need to connect the Bacalhau node to an IPFS server so that you can run jobs that consume CIDs as inputs. 

You can either [install IPFS](https://docs.ipfs.tech/install/) and run it locally, or you can connect to a remote IPFS server. In both cases, you should have an [IPFS multiaddress](https://richardschneider.github.io/net-ipfs-core/articles/multiaddress.html) for the IPFS server that should look something like this:

```bash
ipfs id | grep p2p

"/ip4/127.0.0.1/tcp/5001/p2p/QmVcSqVEsvm5RR9mBLjwpb2XjFVn5bPdPL69mL8PH45pPC"
```

:::caution

The multiaddress above is just an example - you'll need to get the multiaddress of the IPFS server you want to connect to.

:::

You can then configure your Bacalhau node to use this IPFS server by passing the `--ipfs-connect` argument to the `serve` command:

```bash
bacalhau serve --ipfs-connect /ip4/127.0.0.1/tcp/5001/p2p/QmVcSqVEsvm5RR9mBLjwpb2XjFVn5bPdPL69mL8PH45pPC
```

Alternatively, you can set the `Node.IPFS.Connect` property in the Bacalhau configuration file.

### S3

To get data from an S3 source, you need to either make the data public or specify the necessary credentials in your job. See the [S3 source specification](../other-specifications/sources/s3.md#credential-requirements) for more details. See an example of a command in the imperative approach specifying details to get input data from the S3 storage:

```bash
bacalhau docker run -i src=s3://bucket/key,dst=/my/input/path,opt=endpoint=http://s3.example.com,opt=region=us-east-1 ubuntu ...
```

## Publishers

### IPFS

The IPFS publisher works using the same setup as above - you'll need to have an
IPFS server running and a multiaddress for it. Then pass that
multiaddress using the `--ipfs-connect` argument to the `serve` command.

If you are publishing to a public IPFS node, you can use `bacalhau get` with no
further arguments to download the results. However, you may experience a delay
in results becoming available as indexing of new data by public nodes takes
time.

To speed up the download or to retrieve results from a private IPFS node, pass
the swarm multiaddress to `bacalhau get` to download results.

```bash
# Use the --ipfs-swarm-addrs flag,
# or set the Node.IPFS.SwarmAddresses config property.

bacalhau get $JOB_ID --ipfs-swarm-addrs /ip4/.../tcp/5001/p2p/Qmy...
```

Pass the swarm key to `bacalhau get` if the IPFS swarm is a private swarm.

```bash
# Use the --ipfs-swarm-key flag,
# or set the Node.IPFS.SwarmKeyPath config property.

bacalhau get $JOB_ID --ipfs-swarm-key ./path/to/swarm.key
```
### S3

To upload the result of your job to the S3 storage, you will need to provide a name and key for the S3 bucket. See the [S3 publisher specification](../other-specifications/publishers/s3.md) page for more details. See an example of a command in the imperative approach specifying details to publish results in S3:

```bash
bacalhau docker run -p s3://bucket/key,opt=endpoint=http://s3.example.com,opt=region=us-east-1 ubuntu ...
```