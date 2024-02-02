---
sidebar_label: 'Test Network Locally'
sidebar_position: 155
---

# Test Network Locally

Before you join the demo Bacalhau network or any private network, you can test locally.

To test, you can set environment variable `export PREDICTABLE_API_PORT=1` to point devstack to run on port 20000 and execute the `bacalhau devstack` command, which runs locally a cluster of 4 nodes: one requester and 3 compute. 

```bash
export PREDICTABLE_API_PORT=1
bacalhau devstack
```

Now in another terminal tab set the following environment variables to connect your client binary to the local development cluster

```bash
export BACALHAU_API_HOST=127.0.0.1
export BACALHAU_API_PORT=20000
```

Done! You can now interact with Bacalhau. All jobs will be routed to the local cluster and you can see the logs of their reception and execution.

```bash
bacalhau docker run ubuntu echo hello
bacalhau list
```

