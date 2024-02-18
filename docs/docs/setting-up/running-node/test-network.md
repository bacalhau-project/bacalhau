---
sidebar_label: 'Test Network Locally'
sidebar_position: 155
---

# Test Network Locally

Before you join the main Bacalhau network, you can test locally.

To test, you can use the `bacalhau devstack` command, which offers a way to get a 3 node cluster running locally.

```bash
export PREDICTABLE_API_PORT=1
bacalhau devstack
```

:::tip

By settings `PREDICTABLE_API_PORT=1` , the first node of our 3 node cluster will always listen on port **20000**

:::

In another window, export the following environment variables so that the Bacalhau client binary connects to our local development cluster:

```bash
export BACALHAU_API_HOST=127.0.0.1
export BACALHAU_API_PORT=20000
```

You can now interact with Bacalhau - all jobs are running by the local devstack cluster.

```bash
bacalhau docker run ubuntu echo hello
bacalhau list
```
