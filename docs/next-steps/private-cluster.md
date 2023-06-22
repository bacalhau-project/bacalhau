---
sidebar_label: 'Private Cluster'
sidebar_position: 5
---
# Private Cluster

It is possible to run Bacalhau completely disconnected from the main Bacalhau network so that you can run private workloads without risking running on public nodes or inadvertently sharing your data outside of your organization. The isolated network will not connect to the public Bacalhau network nor connect to a public network. To do this, we will run our network in-process rather than externally.

:::info
A private network and storage is easier to set up, but a separate public server is better for production. The private network and storage will use a temporary directory for its repository and so the contents will be lost on shutdown.
:::

## Initial Requester Node

The first step is to start up the initial node, which we will use as the `requester node`. This node will connect to nothing but will listen for connections.

```bash
bacalhau serve --node-type requester --private-internal-ipfs --peer none
```

This will produce output similar to this:

```
16:34:17.154 | INF pkg/libp2p/host.go:69 > started libp2p host [host-id:QmWg7m5GyAhocrd8o18dtntua7dQeEHpuHxC3niRH4pnvE] [listening-addresses:["/ip4/192.168.1.224/tcp/1235","/ip4/127.0.0.1/tcp/1235","/ip4/192.168.1.224/udp/1235/quic","/ip4/127.0.0.1/udp/1235/quic","/ip6/::1/tcp/1235","/ip6/::1/udp/1235/quic"]] [p2p-addresses:["/ip4/192.168.1.224/tcp/1235/p2p/QmWg7m5GyAhocrd8o18dtntua7dQeEHpuHxC3niRH4pnvE","/ip4/127.0.0.1/tcp/1235/p2p/QmWg7m5GyAhocrd8o18dtntua7dQeEHpuHxC3niRH4pnvE","/ip4/192.168.1.224/udp/1235/quic/p2p/QmWg7m5GyAhocrd8o18dtntua7dQeEHpuHxC3niRH4pnvE","/ip4/127.0.0.1/udp/1235/quic/p2p/QmWg7m5GyAhocrd8o18dtntua7dQeEHpuHxC3niRH4pnvE","/ip6/::1/tcp/1235/p2p/QmWg7m5GyAhocrd8o18dtntua7dQeEHpuHxC3niRH4pnvE","/ip6/::1/udp/1235/quic/p2p/QmWg7m5GyAhocrd8o18dtntua7dQeEHpuHxC3niRH4pnvE"]]
16:34:17.555 | INF cmd/bacalhau/serve.go:506 > Internal IPFS node available [NodeID:QmWg7m5G] [ipfs_swarm_addresses:["/ip4/192.168.1.224/tcp/53291/p2p/QmdCLbe2pUoGjCzffd75U8w1LTiVpSap88rNjzXsBhWkL2","/ip4/127.0.0.1/tcp/53291/p2p/QmdCLbe2pUoGjCzffd75U8w1LTiVpSap88rNjzXsBhWkL2"]]
```

To connect another node to this private one, run the following command in your shell:
```
bacalhau serve --private-internal-ipfs --peer /ip4/192.168.1.224/tcp/1235/p2p/QmWg7m5GyAhocrd8o18dtntua7dQeEHpuHxC3niRH4pnvE --ipfs-swarm-addr /ip4/192.168.1.224/tcp/53291/p2p/QmdCLbe2pUoGjCzffd75U8w1LTiVpSap88rNjzXsBhWkL2

To use this requester node from the client, run the following commands in your shell:
export BACALHAU_IPFS_SWARM_ADDRESSES=/ip4/192.168.1.224/tcp/53291/p2p/QmdCLbe2pUoGjCzffd75U8w1LTiVpSap88rNjzXsBhWkL2
export BACALHAU_API_HOST=0.0.0.0
export BACALHAU_API_PORT=1234
```

## Compute Nodes

To connect another node to this private one, run the following command in your shell:

```
bacalhau serve --private-internal-ipfs --peer /ip4/<ip-address>/tcp/1235/p2p/<peer-id> --ipfs-swarm-addr /ip4/<ip-address>/tcp/<port>/p2p/<peer-id>
```

:::tip
The exact command will be different on each computer and is outputted by the `bacalhau serve --node-type requester ...` command
:::

The command `bacalhau serve --private-internal-ipfs --peer ...` starts up a compute node and adds it to the cluster.

## Submitting Jobs

To use this cluster from the client, run the following commands in your shell:

```
export BACALHAU_IPFS_SWARM_ADDRESSES=/ip4/<ip-address>/tcp/<port>/p2p/<peer-id>
export BACALHAU_API_HOST=0.0.0.0
export BACALHAU_API_PORT=1234
```

:::tip
The exact command will be different on each computer and is outputted by the `bacalhau serve --node-type requester ...` command
:::

The command `export BACALHAU_IPFS_SWARM_ADDRESSES=...` sends jobs into the cluster from the command line client.

## Public IPFS Network

Instructions for connecting to the public IPFS network via the private Bacalhau cluster:

On all nodes, start ipfs:

```
ipfs init
```
Then run the following command in your shell:

```
export IPFS_CONNECT=$(ipfs id |grep tcp |grep 127.0.0.1 |sed s/4001/5001/|sed s/,//g |sed 's/"//g')
```

On the **first node** execute the following:

```
export LOG_LEVEL=debug
bacalhau serve --peer none --ipfs-connect $IPFS_CONNECT --node-type requester,compute
```
Monitor the output log for:
`11:16:03.827 | DBG pkg/transport/bprotocol/compute_handler.go:39 > ComputeHandler started on host QmWXAaSHbbP7mU4GrqDhkgUkX9EscfAHPMCHbrBSUi4A35`


On **all other nodes** execute the following:

```
export PEER_ADDR=/ip4/<public-ip>/tcp/1235/p2p/<above>
````
Replace the values in the command above with your own value

Here is our example:

```
export PEER_ADDR=/ip4/192.18.129.124/tcp/1235/p2p/QmWXAaSHbbP7mU4GrqDhkgUkX9EscfAHPMCHbrBSUi4A35
bacalhau serve --peer $PEER_ADDR --ipfs-connect $IPFS_CONNECT --node-type compute
```

Then from any client set the following before invoking your Bacalhau job:

```
export BACALHAU_API_HOST=address-of-first-node
```

## Deploy a private cluster

A private cluster is a network of Bacalhau nodes completely isolated from any public node.
That means you can safely process private jobs and data on your cloud or on-premise hosts!

Good news. Spinning up a private cluster is really a piece of cake :cake::

1. Install Bacalhau `curl -sL https://get.bacalhau.org/install.sh | bash` on every host
1. Run `bacalhau serve` only on one host, this will be our "bootstrap" machine
1. Copy and paste the command it outputs under the "*To connect another node to this private one, run the following command in your shell...*" line to the **other hosts**
1. Copy and paste the env vars it outputs under the "*To use this requester node from the client, run the following commands in your shell...*" line to a **client machine**
1. Run `bacalhau docker run ubuntu echo hello` on the client machine
1. That's all folks! :tada:

Optionally, set up [systemd](https://en.wikipedia.org/wiki/Systemd) units make Bacalhau daemons permanent, here's an example [systemd service file](https://github.com/bacalhau-project/bacalhau/blob/main/ops/terraform/remote_files/configs/bacalhau.service).

Please contact us on [Slack](https://bit.ly/bacalhau-project-slack/) `#bacalhau` channel for questions and feedback!
