---
sidebar_label: 'Fundamentals'
sidebar_position: 1
---

# Networking Fundamentals in Bacalhau

Bacalhau uses [libp2p](https://libp2p.io/) under the hood to communicate with other nodes on the network.

## Peer identity

Because bacalhau is built using libp2p, the concept of peer identity is used to identify nodes on the network.

Bacalhau uses 2 keys to identify the node and the user:
- A private Ed25519 `libp2p_private_key` key is used to get the Peer ID in libp2p
- A private RSA key in the PKCS1 format `user_id.pem` is used to uniquely identify user on the network when making reqeusts from CLI

By default, Bacalhau creates its own keys on installation. These keys can be found at the path given by the config:
- key `User.KeyPath` for a RSA key, which by default is `~/.bacalhau/user_id.pem`
- key `User.Libp2pkeypath` for an Ed25519 key, which by default is `~/.bacalhau/libp2p_private_key`


When you start a bacalhau node using `bacalhau serve`, it will look for these keys in the paths, specified in the environmental variable or config. If it doesn't find them, it will generate a new ones and save them there.

:::info
The following source prioritization applies to both this and the rest of the application settings:
1. Flags
2. Environmental variables
3. Config file
4. Default config
:::

You can override the directory where the private key is stored using the `BACALHAU_DIR` environment variable.

The peer identity is derived from the private key and is used to identify the node on the network. You can get the peer identity of a node by running `bacalhau id`:

```bash
bacalhau id
```
Sample output:

```bash
{"ID":"QmPmaoCVvMsQ8xHGdM7RBtVw5PDroxCSe7iAeS75nDLHu2","ClientID":"051d3d9f30e64f3fe42b0ff6d6f576f7b680a8ebd8b4ca59f60a1fa90f0d808b"}
```
## Configure peers

By default, running `bacalhau serve` will connect to the public bootstrap nodes, which can be found in the config file by the key `Node.BootstrapAddress`

Bacalhau uses libp2p [multiaddresses](https://docs.libp2p.io/concepts/addressing/) to identify nodes on the network.

If you want to connect to other nodes, you need to know their Peer IDs and use the `--peer` flag to specify additional peers to connect to (comma-separated list). Here is the sample command which starts and connects the node to 2 other nodes:

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

:::info
In the libp2p the term swarm has been replaced by [switch](https://docs.libp2p.io/concepts/appendix/glossary/#switch), however swarm is still used for historical reasons 
:::

## REST API port

The Bacalhau node exposes a REST API that can be used to query the node for information. You can browse up-tp-date [API documentation](https://github.com/bacalhau-project/bacalhau/blob/main/docs/swagger.json) via the [Swagger Editor](https://editor.swagger.io/?url=https://raw.githubusercontent.com/bacalhau-project/bacalhau/main/docs/swagger.json).

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

## Using the API

You can call the node API to get information from it. For example:

```bash
curl localhost:1234/api/v1/id
```

Once you run the command above, you'll get a node ID, same as from `bacalhau id`:

```bash
"QmPmaoCVvMsQ8xHGdM7RBtVw5PDroxCSe7iAeS75nDLHu2"
```
