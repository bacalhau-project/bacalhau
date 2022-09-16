---
sidebar_label: 'Networking'
sidebar_position: 120
---

# Networking

Bacalhau uses [libp2p](https://libp2p.io/) under the hood to communicate with other nodes on the network.

## Configure peers

By default - running `bacalhau serve` will connect to the following nodes (which are thge default bootstrap nodes run by Protocol labs):

```
/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF
/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
```

If you want to connect to other nodes - you can use the `--peer` flag to specify additional peers to connect to (comma separated list).

```bash
bacalhau serve \
  --peer /ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL,/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF
```

## libp2p swarm port

The default port the libp2p swarm listens on is **1235**.

You can configure the swarm port using the `--port` flag:

```bash
bacalhau serve \
  --port 1235
```

To ensure that our node can communicate with other nodes on the network - we need to make sure the swarm port is open and accesible by other nodes.

## REST api port

Our bacalhau node exposes a REST api that can be used to query the node for information.

The default port the REST api listens on is **1234**.

The default network interface the REST api listens on is **0.0.0.0**.

You can configure the REST api port using the `--api-port` flag:

You can also configure which network interface the REST api will bind to using the `--host` flag:

```bash
bacalhau serve \
  --api-port 1234 \
  --host 127.0.0.1
```

:::tip

You can use the `--host` flag to restrict network access to the REST api.

:::
