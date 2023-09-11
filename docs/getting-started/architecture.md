---
sidebar_label: 'Architecture'
sidebar_position: 1
---

# Architecture

Bacalhau is a peer-to-peer network of nodes that allows for decentralized communication between computers. There are two nodes types in the network:
- **Requester Node:** responsible for handling user requests, discovering and ranking compute nodes, forwarding jobs to compute nodes, and monitoring the job lifecycle.
- **Compute Node:** responsible for executing jobs and producing results. Different compute nodes can be used for different types of jobs, depending on their capabilities and resources.

![image](../../static/img/architecture/architecture-purpose.jpg 'Bacalhau Architecture')

The requester and compute nodes together form a p2p network and use gossiping to discover each other, share information about node capabilities, available resources and health status.

To interact with the Bacalhau network, users can use the Bacalhau CLI (command-line interface) to send requests to a requester node in the network. These requests are sent using the JSON format over HTTP, a widely-used protocol for transmitting data over the internet.

## System Components

Bacalhau's architecture involves two main sections which are the **core components** and **interfaces**.

## Core Components

The core components are responsible for handling requests and connecting different nodes. It consists of:

- [Requester node](#requester-node)
- [Compute node](#compute-node)

### Requester node

In the Bacalhau network, the requester node is responsible for handling requests from clients using JSON over HTTP. This node serves as the main custodian of jobs that are submitted to it.

When a job is submitted to a requester node, it selects compute nodes that are capable and suitable to execute the job, and communicate with them directly. It is important to note that there is only ever a single requester node for a given job, which is the node that the job was originally submitted to.

Overall, the requester node plays a crucial role in the Bacalhau network, serving as the main point of contact for clients and the primary handler of jobs that are submitted to the network.

### Compute node

In the Bacalhau network, the compute node plays a critical role in the process of executing jobs and producing results. While the requester does its best to be up-to-date with the network status, it is the compute node that is responsible for determining whether it can execute a job or not. This model allows for a more decentralized approach to job orchestration as the network will function properly even if the requester nodes have stale view of the network, or if concurrent requesters are allocating jobs to the same compute nodes. 

Once the compute node has run the job and produced results, it will publish the results to a remote destination as specified in the job specification (e.g. S3), and notify the requester of the job completion. The compute node has a collection of named executors, storage sources, and publishers, and it will choose the most appropriate ones based on the job specifications.

## Interface

The interface handles the distribution, execution, storage and publishing of jobs.

- [Transport](#transport)
- [Executor](#executor)
- [Storage Provider](#storage-provider)
- [Verifier](#verifier)
- [Publisher](#publisher)

### Transport

The transport interface is responsible for sending messages about jobs that are created, accepted, and executed  to other compute nodes. It also manages the identity of individual Bacalhau nodes to ensure that messages are only delivered to authorized nodes, which improves network security.

To achieve this, the transport interface uses a protocol called **bprotocol**, which is a point-to-point scheduling protocol that runs over [libp2p](https://libp2p.io/) and is used to distribute job messages efficiently to other nodes on the network. This is our upgrade to the [GossipSub](https://docs.libp2p.io/concepts/publish-subscribe/) handler as it ensures that messages are delivered to the right nodes without causing network congestion, thereby making communication between nodes more scalable and efficient.

### Executor

The executor is a critical component of the Bacalhau network that handles the execution of jobs and ensures that the storage used by the job is local. One of its main responsibilities is to present the input and output storage volumes into the job when it is run.

The executor performs two primary functions: 
- presenting the storage volumes in a format that is suitable for the executor, and,
- running the job.

When the job is completed, the executor will merge the stdout, stderr, and named output volumes into a results folder that is then published to a remote location.

Overall, the executor plays a crucial role in the Bacalhau network by ensuring that jobs are executed properly, and their results are published accurately.

### Storage Provider

In a peer-to-peer network like Bacalhau, storage providers play a crucial role in presenting an upstream storage source. There can be different storage providers available in the network, each with its own way of manifesting the CID (Content IDentifier) to the executor.

For instance, there can be a POSIX storage provider that presents the CID as a POSIX filesystem, or a library storage provider that streams the contents of the CID via a library call.

Therefore, the storage providers and Executor implementations are loosely coupled, allowing the POSIX and library storage providers to be used across multiple executors, wherever it is deemed appropriate.

### Publisher

The publisher is responsible for uploading the final results of a job to a remote location where clients can access them, such as S3 or IPFS

## Job Lifecycle

The job lifecycle involves several steps that are handled by different components of the network, from job submission to job download.

### Job Submission

Jobs submitted via the Bacalhau CLI are forwarded to a Bacalhau network node at `bootstrap.production.bacalhau.org` via port 1234 by default. This Bacalhau node will act as the requester node for the duration of the job lifecycle. Jobs can also be submitted to any requester node on the Bacalhau network.

When jobs are submitted to the requester node, the requester will select few compute nodes that are capable to execute the job and ask them to run. The job will have a `concurrency` setting, which refers to how many different nodes you may want to run this job.  

The job might also mention the use of `volumes` (for example some CIDs). The compute node can choose to bid on the job if the data for the volume resides locally in the compute node, or it can choose to bid anyway. Bacalhau supports the use of external HTTP or exec hooks to decide if a node wants to bid on a job. This means that a node operator can give granular rules about the jobs they are willing to run.


### Job Acceptance

As bids from compute nodes arrive back at the originating requester node, it can choose which bids to accept and which ones to reject. This can be based on the previous reputation of each compute node or any other factors the requester node might take into account (like locality, hardware resources, cost etc). The requester node will also have the same http or exec hooks to decide if it wants to accept a bid from a given compute node. 


### Job Execution

As accepted bids are received by compute nodes, they will `execute` the job using the executor for that job, and the storage providers that the executor has mapped in.

For example, a job could use the `docker` executor, `WASM` executor, or a library storage volumes. This would result in a POSIX mount of the storage into a running container or a WASM-style `syscall` to stream the storage bytes into the WASM runtime. Each executor will deal with storage in a different way, so even though each job mentions the storage volumes, they would both end up with different implementations at runtime.



### Publishing

Once results are ready, the publisher will publish the raw results folder currently residing on the compute node. The publisher interface mainly consists of a single function, which has the task of uploading the local results folder somewhere and returning a storage reference to where it has been uploaded.


### Networking

It is possible to run Bacalhau completely disconnected from the main Bacalhau network so that you can run private workloads without risking running on public nodes or inadvertently sharing your data outside of your organization. The isolated network will not connect to the public Bacalhau network nor connect to a public network. Read more on [networking](https://docs.bacalhau.org/next-steps/private-cluster)


### Input / Output Volumes

A job includes the concept of input and output volumes, and the Docker executor implements support for these. This means you can specify your CIDs, URLs, and/or S3 objects as input paths and also write results to an output volume. This can be seen in the following example:

```
bacalhau docker run \
  -i s3://mybucket/logs-2023-04*:/input \
  -o apples:/output_folder \
  ubuntu \
  bash -c 'ls /input > /output_folder/file.txt'
```

The above example demonstrates an input volume flag `-i s3://mybucket/logs-2023-04*`, which mounts all S3 objects in bucket `mybucket` with `logs-2023-04` prefix within the docker container at location `/input` (root).

Output volumes are mounted to the Docker container at the location specified. In the example above, any content written to `/output_folder` will be made available within the "apples" folder in the job results CID.

Once the job has run on the executor, the contents of `stdout` and `stderr` will be added to any named output volumes the job has used (in this case apples), and all those entities will be packaged into the results folder which is then published to a remote location by the publisher.
