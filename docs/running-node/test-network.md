---
sidebar_label: 'Test Network'
sidebar_position: 115
---

# Test Network

Before you join the main bacalhau network - you can try things out locally first.

To do this we can use the `bacalhau devstack` command which is a quick way to get a 3 node cluster running locally.

```bash
export PREDICTABLE_API_PORT=1
bacalhau devstack
```

:::tip

By settings `PREDICTABLE_API_PORT=1` - it means the first node of our 3 node cluster will always listen on port **20000**

:::

In another window - we can now export the following environment variables so our bacalhau client binary will connect to our local development cluster:

```bash
export BACALHAU_API_HOST=127.0.0.1
export BACALHAU_API_PORT=20000
```

We can now interact with bacalhau as normal - all jobs are being run by our local devstack cluster.

```bash
bacalhau docker run ubuntu echo hello
bacalhau list
```

