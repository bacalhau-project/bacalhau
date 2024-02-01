---
sidebar_label: 'Fundamentals'
sidebar_position: 1
---

# Networking Fundamentals in Bacalhau

Bacalhau uses [libp2p](https://libp2p.io/) under the hood to communicate with other nodes on the network.

## Peer identity

Because bacalhau is built using libp2p, the concept of peer identity is used to identify nodes on the network.

When you start a bacalhau node using `bacalhau serve`, it will look for an RSA private key in the `~/.bacalhau` directory. If it doesn't find one, it will generate a new one and save it there.

You can override the directory where the private key is stored using the `BACALHAU_PATH` environment variable.

Private keys are named after the port used for the libp2p connection which defaults to `1235`. By default when first starting a node, the private key will be stored in `~/.bacalhau/private_key.1235`.

The peer identity is derived from the private key and is used to identify the node on the network. You can get the peer identity of a node by running `bacalhau id`:

```bash
bacalhau id
```

## Configure peers

By default, running `bacalhau serve` will connect to the following nodes (which are the default bootstrap nodes run by Protocol labs):

```
/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF
/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
```

Bacalhau uses libp2p [multiaddresses](https://docs.libp2p.io/concepts/addressing/) to identify nodes on the network.

If you want to connect to other nodes, and you know their Peer IDs you can use the `--peer` flag to specify additional peers to connect to (comma-separated list).

```bash
bacalhau serve \
  --peer /ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL,/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF
```

If you want to connect to a requester node, and you know it's IP but not it's Peer ID, you can use the following which will contact the requester API directly and ask for the current Peer ID instead.

```bash
bacalhau serve \
  --peer /ip4/35.245.115.191/tcp/1234/http
```

## libp2p swarm port

The default port the libp2p swarm listens on is **1235**.

You can configure the swarm port using the `--port` flag:

```bash
bacalhau serve \
  --port 1235
```

To ensure that the node can communicate with other nodes on the network, make sure the swarm port is open and accessible by other nodes.

## REST API port

The Bacalhau node exposes a REST API that can be used to query the node for information.

The default port the REST API listens on is **1234**.

The default network interface the REST API listens on is **0.0.0.0**.

You can configure the REST API port using the `--api-port` flag:

You can also configure which network interface the REST API will bind to using the `--host` flag:

```bash
bacalhau serve \
  --api-port 1234 \
  --host 127.0.0.1
```

:::tip

You can use the `--host` flag to restrict network access to the REST API.

:::

## Generic Endpoint

You can call [http://dashboard.bacalhau.org:1000/api/v1/run](http://dashboard.bacalhau.org:1000/api/v1/run) with the POST body as a JSON serialized spec

```bash
curl -XPOST -d '{"Engine": "Docker", "Docker": {"Image": "ubuntu", "Entrypoint": ["echo"], "Parameters": ["hello"]}, "Deal": {"Concurrency": 1}, "Verifier": "Noop", "PublisherSpec":{"Type":"IPFS"}}' 'http://dashboard.bacalhau.org:1000/api/v1/run';
```

Once you run the command above, you'll get a CID output:

```bash
"cid": "QmeNhAA97qtdGHQtd1Qvgk13C6GHkn6aTCT8ih53JLN7vL"
```
