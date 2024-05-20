# Node Management

Bacalhau clusters are composed of requester nodes, and compute nodes. The requester nodes are responsible for managing the compute nodes that make up the cluster. This functionality is only currently available when using NATS for the network transport.

The two main areas of functionality for the requester nodes are, managing the membership of compute nodes that require approval to take part in the cluster, and monitoring the health of the compute nodes. They are also responsible for collecting information provided by the compute nodes on a regular schedule.

## Compute node membership

As compute nodes start, they register their existence with the requester nodes. Once registered, they will maintain a sentinel file to note that they are already registered, this avoids unnecessary registration attempts.

Once registered, the requester node will need to approve the compute node before it can take part in the cluster. This is to ensure that the requester node is aware of all the compute nodes that are part of the cluster. In future, we may provide mechanisms for auto-approval of nodes joining the cluster, but currently all compute nodes registering default to the PENDING state.

Listing the current nodes in the system will show requester nodes automatically APPROVED, and compute nodes in the PENDING state.

```shell
$ bacalhau node list # extra columns removed

ID      TYPE       APPROVAL  STATUS
node-0  Requester  APPROVED  UNKNOWN
node-1  Compute    PENDING   HEALTHY
node-2  Compute    PENDING   HEALTHY
node-3  Compute    PENDING   HEALTHY
```

Nodes can be rejected using their node id, and optionally specifying a reason with the -m flag.

```shell
$ bacalhau node reject node-3 -m "malicious node?"
Ok
```

Nodes can be approved using their node id.

```shell
$ bacalhau node approve node-1
Ok
```

There is currently no support for auto-eviction of nodes, but they can be manually removed from the cluster using the `node delete` command. Note, if
they are manually removed, they are able to manually re-register, so this is
most useful when you know the node will not be coming back.

```shell
$ bacalhau node delete node-2
```

After all of these actions, the node list looks like

```shell
$ bacalhau node list # extra columns removed

ID      TYPE       APPROVAL  STATUS
node-0  Requester  APPROVED  UNKNOWN
node-1  Compute    APPROVED  HEALTHY
node-3  Compute    REJECTED  HEALTHY
```

## Compute node updates

Compute nodes will provide information about themselves to the requester nodes on a regular schedule. This information is used to help the requester nodes make decisions about where to schedule workloads.

These updates are broken down into:

- **Node Information**: This is the information about the node itself, such as the hostname, CPU architecture, and any labels associated with the node. This information is persisted to the Node Info Store.
- **Resource Information**: This is the information about the resources available on the node, such as the amount of memory, storage and CPU available. This information is held in memory and used to make scheduling decisions. It is not persisted to disk as it is considered transient.
- **Health Information**: This heartbeat is used to determine if the node is still healthy, and if it is not, the requester node will mark the node as unhealthy. Eventually, the node will be marked as Unknown if it does not recover. This information is held in memory and used to make scheduling decisions. Like the resource information, it is not persisted to disk as it is considered transient.

Various configuration options are available to control the frequency of these updates, and the timeout for the health check. These can be set in the configuration file.

For the compute node, these settings are:

- **Node Information**: `InfoUpdateFrequency` - The interval between updates of the node information.

- **Resource Information**: `ResourceUpdateFrequency` - The interval between updates of the resource information.

- **Heartbeat**: `HeartbeatFrequency` - The interval between heartbeats sent by the compute node.

- **Heartbeat**: `HeartbeatTopic` - The name of the pubsub topic that heartbeat messages are sent via.

For the requester node, these settings are:

- **Heartbeat** `HeartbeatFrequency` - How often the heartbeat server will check the priority queue of node heartbeats.

- **Heartbeat** `HeartbeatTopic` - The name of the pubsub topic that heartbeat messages are sent via. Should be the same as the compute node value.

- **Node health** `NodeDisconnectedAfter` - The interval after which the node will be considered disconnected if a heartbeat has not been received.

## Cluster membership events

As compute nodes are added and removed from the cluster, the requester nodes will emit events to the NATS PubSub system. These events can be consumed by other systems to react to changes in the cluster membership.

```

```
