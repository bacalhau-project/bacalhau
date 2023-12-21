---
sidebar_label: 'How Bacalhau Works'
sidebar_position: 1
---

# How Bacalhau Works

## Job preparation

Define and create jobs in the Bacalhau network, leveraging job types introduced in v1.1 for enhanced orchestration and scheduling. Explore provided [Job Types].

Jobs may include parameters, resource requirements, and data details. Check the [Job Specification] for information on job and server-generated parameters and [task execution specifics].

Prepare data with Bacalhau by [copying from URLs], [pinning to public storage], or [copying from an S3 bucket]. Mount data anywhere for Bacalhau to run against. Refer to [IPFS], [Local], [S3], and [URL] Source Specifications for data source usage.

Optimize workflows without completely redesigning them. Run arbitrary tasks using Docker containers and WebAssembly images. Follow the Onboarding guides for [Docker] and [WebAssembly] workloads.

Explore GPU workload support with Bacalhau. Learn how to run GPU workloads using the Bacalhau client in the [GPU Workloads] section.

Integrate Python applications with Bacalhau using the [Bacalhau Python SDK].

For node operation, refer to the [Running a Node] section for configuring and running a Bacalhau node.

If you prefer an isolated environment, explore the [Private Cluster] for performing tasks without connecting to the main Bacalhau network.

## Architecture

Bacalhau is a peer-to-peer network of nodes that enables decentralized communication between computers. The network consists of two types of nodes:  

  **Requester Node:** responsible for handling user requests, discovering and ranking compute nodes, forwarding jobs to compute nodes, and monitoring the job lifecycle.  

  **Compute Node:** responsible for executing jobs and producing results. Different compute nodes can be used for different types of jobs, depending on their capabilities and resources.

![image](../../static/img/architecture/architecture-purpose.jpg 'Bacalhau Architecture')

The requester and compute nodes together form a p2p network and use gossiping to discover each other, share information about node capabilities, available resources and health status.

To interact with the Bacalhau network, users can use the Bacalhau CLI (command-line interface) to send requests to a requester node in the network. These requests are sent using the JSON format over HTTP, a widely-used protocol for transmitting data over the internet.

Bacalhau's architecture involves two main sections which are the **core components** and **interfaces**.

![image](../../static/img/architecture/System-Components.png 'System-Components')


### 1. Core Components

The core components are responsible for handling requests and connecting different nodes. The section includes:

  [Requester node](#requester-node)  
  [Compute node](#compute-node)

### Requester node 

In the Bacalhau network, the requester node is responsible for handling requests from clients using JSON over HTTP. This node serves as the main custodian of jobs that are submitted to it.

When a job is submitted to a requester node, it selects compute nodes that are capable and suitable to execute the job, and communicate with them directly. It is important to note that there is only ever a single requester node for a given job, which is the node that the job was originally submitted to.

Overall, the requester node plays a crucial role in the Bacalhau network, serving as the main point of contact for clients and the primary handler of jobs that are submitted to the network.

### Compute node

In the Bacalhau network, the compute node plays a critical role in the process of executing jobs and producing results. While the requester does its best to be up-to-date with the network status, it is the compute node that is responsible for determining whether it can execute a job or not. This model allows for a more decentralized approach to job orchestration as the network will function properly even if the requester nodes have stale view of the network, or if concurrent requesters are allocating jobs to the same compute nodes. 

Once the compute node has run the job and produced results, it will publish the results to a remote destination as specified in the job specification (e.g. S3), and notify the requester of the job completion. The compute node has a collection of named executors, storage sources, and publishers, and it will choose the most appropriate ones based on the job specifications.

### 2. Interface 

The interface handles the distribution, execution, storage and publishing of jobs.

  [Transport](#transport)  
  [Executor](#executor)  
  [Storage Provider](#storage-provider)    
  [Publisher](#publisher)  

### Transport

The transport interface is responsible for sending messages about jobs that are created, accepted, and executed  to other compute nodes. It also manages the identity of individual Bacalhau nodes to ensure that messages are only delivered to authorized nodes, which improves network security.

To achieve this, the transport interface uses a protocol called **bprotocol**, which is a point-to-point scheduling protocol that runs over [libp2p](https://libp2p.io/) and is used to distribute job messages efficiently to other nodes on the network. This is our upgrade to the [GossipSub](https://docs.libp2p.io/concepts/publish-subscribe/) handler as it ensures that messages are delivered to the right nodes without causing network congestion, thereby making communication between nodes more scalable and efficient.

### Executor

The executor is a critical component of the Bacalhau network that handles the execution of jobs and ensures that the storage used by the job is local. One of its main responsibilities is to present the input and output storage volumes into the job when it is run.

The executor performs two primary functions: presenting the storage volumes in a format that is suitable for the executor and running the job.

When the job is completed, the executor will merge the stdout, stderr and named output volumes into a results folder that is then published to a remote location.

Overall, the executor plays a crucial role in the Bacalhau network by ensuring that jobs are executed properly, and their results are published accurately.

### Storage Provider

In a peer-to-peer network like Bacalhau, storage providers play a crucial role in presenting an upstream storage source. There can be different storage providers available in the network, each with its own way of manifesting the CID (Content IDentifier) to the executor.

For instance, there can be a POSIX storage provider that presents the CID as a POSIX filesystem, or a library storage provider that streams the contents of the CID via a library call.

Therefore, the storage providers and Executor implementations are loosely coupled, allowing the POSIX and library storage providers to be used across multiple executors, wherever it is deemed appropriate.

### Publisher

The publisher is responsible for uploading the final results of a job to a remote location where clients can access them, such as S3 or IPFS.

## Job Submission

You should use the Bacalhau client to send a task to the network.
The client transmits the job information to the Bacalhau network via established protocols and interfaces. 
Jobs submitted via the Bacalhau CLI are forwarded to a Bacalhau network node at http://bootstrap.production.bacalhau.org/ via port 1234 by default. This Bacalhau node will act as the requester node for the duration of the job lifecycle.

Bacalhau provides an interface to interact with the server via a REST API. Bacalhau uses 127.0.0.1 as the localhost and 1234 as the port by default.

### Create a Job

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs
defaultValue="CLI"
values={[
{label: 'CLI', value: 'CLI'},
{label: 'API', value: 'API'},
]}>
<TabItem value="CLI">

    `bacalhau create [flags]`

You can use the command with [appropriate flags] to create a job in Bacalhau using JSON and YAML formats.

</TabItem>
<TabItem value="API">

    Endpoint: `PUT /api/v1/orchestrator/jobs`

    You can use [Create Job API Documentation] to submit a new job for execution. 

</TabItem>
</Tabs>


 
You can use the `bacalhau docker run` [command] to start a job in a Docker container:

```shell
Usage:
  bacalhau docker run [flags] IMAGE[:TAG|@DIGEST] [COMMAND] [ARG...]

  Flags:
  -c, --concurrency int                  How many nodes should run the job (default 1)
      --confidence int                   The minimum number of nodes that must agree on a verification result
      --cpu string                       Job CPU cores (e.g. 500m, 2, 8).
      --domain stringArray               Domain(s) that the job needs to access (for HTTP networking)
      --download                         Should we download the results once the job is complete?
      --download-timeout-secs duration   Timeout duration for IPFS downloads. (default 5m0s)
      --dry-run                          Do not submit the job, but instead print out what will be submitted
      --engine string                    What executor engine to use to run the job (default "docker")
  -e, --env strings                      The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)
      --filplus                          Mark the job as a candidate for moderation for FIL+ rewards.
  -f, --follow                           When specified will follow the output from the job as it runs
  -g, --gettimeout int                   Timeout for getting the results of a job in --wait (default 10)
      --gpu string                       Job GPU requirement (e.g. 1, 2, 8).
  -h, --help                             help for run
      --id-only                          Print out only the Job ID on successful submission.
  -i, --input storage                    Mount URIs as inputs to the job. Can be specified multiple times. Format: src=URI,dst=PATH[,opt=key=value]
                                         Examples:
                                         # Mount IPFS CID to /inputs directory
                                         -i ipfs://QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72

                                         # Mount S3 object to a specific path
                                         -i s3://bucket/key,dst=/my/input/path

                                         # Mount S3 object with specific endpoint and region
                                         -i src=s3://bucket/key,dst=/my/input/path,opt=endpoint=https://s3.example.com,opt=region=us-east-1

      --ipfs-swarm-addrs string          Comma-separated list of IPFS nodes to connect to. (default "/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL,/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF,/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3")
  -l, --labels strings                   List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.
      --local                            Run the job locally. Docker is required
      --memory string                    Job Memory requirement (e.g. 500Mb, 2Gb, 8Gb).
      --min-bids int                     Minimum number of bids that must be received before concurrency-many bids will be accepted (at random)
      --network network-type             Networking capability required by the job (default None)
      --node-details                     Print out details of all nodes (overridden by --id-only).
      --output-dir string                Directory to write the output to.
  -o, --output-volumes strings           name:path of the output data volumes. 'outputs:/outputs' is always added.
  -p, --publisher publisher              Where to publish the result of the job (default Estuary)
      --raw                              Download raw result CIDs instead of merging multiple CIDs into a single result
  -s, --selector string                  Selector (label query) to filter nodes on which this job can be executed, supports '=', '==', and '!='.(e.g. -s key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.
      --skip-syntax-checking             Skip having 'shellchecker' verify syntax of the command
      --timeout float                    Job execution timeout in seconds (e.g. 300 for 5 minutes and 0.1 for 100ms) (default 1800)
      --verifier string                  What verification engine to use to run the job (default "noop")
      --wait                             Wait for the job to finish. (default true)
      --wait-timeout-secs int            When using --wait, how many seconds to wait for the job to complete before giving up. (default 600)
  -w, --workdir string                   Working directory inside the container. Overrides the working directory shipped with the image (e.g. via WORKDIR in Dockerfile).
```


You can use the `bacalhau run python` [command] to run a job in Python:

```shell
Usage:
  bacalhau run python [flags]

  Flags:
  -c, --command string                   Program passed in as string (like python)
      --concurrency int                  How many nodes should run the job (default 1)
      --confidence int                   The minimum number of nodes that must agree on a verification result
      --context-path string              Path to context (e.g. python code) to send to server (via public IPFS network) for execution (max 10MiB). Set to empty string to disable (default ".")
      --deterministic                    Enforce determinism: run job in a single-threaded wasm runtime with no sources of entropy. NB: this will make the python runtime executein an environment where only some libraries are supported, see https://pyodide.org/en/stable/usage/packages-in-pyodide.html (default true)
      --download                         Should we download the results once the job is complete?
      --download-timeout-secs duration   Timeout duration for IPFS downloads. (default 5m0s)
  -e, --env strings                      The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)
  -f, --follow                           When specified will follow the output from the job as it runs
  -g, --gettimeout int                   Timeout for getting the results of a job in --wait (default 10)
  -h, --help                             help for python
      --id-only                          Print out only the Job ID on successful submission.
  -i, --input storage                    Mount URIs as inputs to the job. Can be specified multiple times. Format: src=URI,dst=PATH[,opt=key=value]
                                         Examples:
                                         # Mount IPFS CID to /inputs directory
                                         -i ipfs://QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72

                                         # Mount S3 object to a specific path
                                         -i s3://bucket/key,dst=/my/input/path

                                         # Mount S3 object with specific endpoint and region
                                         -i src=s3://bucket/key,dst=/my/input/path,opt=endpoint=https://s3.example.com,opt=region=us-east-1

      --ipfs-swarm-addrs string          Comma-separated list of IPFS nodes to connect to. (default "/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL,/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF,/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3")
  -l, --labels strings                   List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.
      --local                            Run the job locally. Docker is required
      --min-bids int                     Minimum number of bids that must be received before concurrency-many bids will be accepted (at random)
      --node-details                     Print out details of all nodes (overridden by --id-only).
      --output-dir string                Directory to write the output to.
  -o, --output-volumes strings           name:path of the output data volumes
      --raw                              Download raw result CIDs instead of merging multiple CIDs into a single result
  -r, --requirement string               Install from the given requirements file. (like pip)
      --timeout float                    Job execution timeout in seconds (e.g. 300 for 5 minutes and 0.1 for 100ms) (default 1800)
      --wait                             Wait for the job to finish. (default true)
      --wait-timeout-secs int            When using --wait, how many seconds to wait for the job to complete before giving up. (default 600)
```

You can use the `bacalhau wasm run` [command] to run a job compiled into the (WASM) format. 

## Job Acceptance

When a job is submitted to a requester node, it selects compute nodes that are capable and suitable to execute the job, and communicate with them directly. The compute node has a collection of named executors, storage sources, and publishers, and it will choose the most appropriate ones based on the job specifications. 

## Job execution

The selected compute node receives the job and starts its execution inside a container. The container can use different executors to work with the data and perform the necessary actions.  A job can use the docker executor, WASM executor or a library storage volumes. Use [Docker Engine Specification] to view the parameters to configure the Docker Engine. If you want tasks to be executed in a WebAssembly environment, pay attention to [WebAssembly Engine Specification].

## Results publishing

When the Compute node completes the job, it publishes the results to **S3's remote storage**, **IPFS**.

Bacalhau's seamless integration with IPFS ensures that users have a decentralized option for publishing their task results, enhancing accessibility and resilience while reducing dependence on a single point of failure. View [IPFS Publisher Specification] to get the detailed information.

Bacalhau's S3 Publisher provides users with a secure and efficient method to publish task results to any S3-compatible storage service. This publisher supports not just AWS S3, but other S3-compatible services offered by cloud providers like Google Cloud Storage and Azure Blob Storage, as well as open-source options like MinIO. View [S3Publisher Specification] to get the detailed information.

## Returning Information to the Bacalhau Client

The Bacalhau client receives updates on the task execution status and results. A user can access the results and manage tasks through the command line interface.

### Get Job Results

To Get the results of a job you can run `bacalhau get [id] [flags]`

```shell
Usage:
  bacalhau get [id] [flags]

Flags:
      --download-timeout-secs int   Timeout duration for IPFS downloads. (default 600)
  -h, --help                        help for get
      --ipfs-swarm-addrs string     Comma-separated list of IPFS nodes to connect to.
      --output-dir string           Directory to write the output to. (default ".")
```

### Describe a Job

<Tabs
defaultValue="CLI"
values={[
{label: 'CLI', value: 'CLI'},
{label: 'API', value: 'API'},
]}>
<TabItem value="CLI">

    `bacalhau describe [id] [flags]`

    You can use the command with [appropriate flags] to get a full description of a job in yaml format.

</TabItem>
<TabItem value="API">

    Endpoint: `GET /api/v1/orchestrator/jobs/:jobID`

    You can use [Describe Job API Documentation] to retrieve the specification and current status of a particular job. 

</TabItem>
</Tabs>


### List of Jobs

<Tabs
defaultValue="CLI"
values={[
{label: 'CLI', value: 'CLI'},
{label: 'API', value: 'API'},
]}>
<TabItem value="CLI">

    `bacalhau list [flags]`

    You can use the command with [appropriate flags] to list jobs on the network in yaml format.

</TabItem>
<TabItem value="API">

    Endpoint: `GET /api/v1/orchestrator/jobs`

    You can use [List Jobs API Documentation] to retrieve a list of jobs.

</TabItem>
</Tabs>

 
### Job Executions

<Tabs
defaultValue="CLI"
values={[
{label: 'CLI', value: 'CLI'},
{label: 'API', value: 'API'},
]}>
<TabItem value="CLI">

    `bacalhau job executions [id] [flags]`

    You can use the command with [appropriate flags] to list all executions associated with a job, identified by its ID, in yaml format.

</TabItem>
<TabItem value="API">

    Endpoint: `GET /api/v1/orchestrator/jobs/:jobID/executions`

    You can use [Job Executions API Documentation] to retrieve all executions for a particular job.

</TabItem>
</Tabs>

 
## Monitoring and Management

The Bacalhau client provides the user with tools to monitor and manage the execution of jobs. You can get information about status, progress and decide on next steps.
View the  [Bacalhau Agent APIs] if you want to know the node's health, capabilities, and deployed Bacalhau version. 
To get information about the status and characteristics of the nodes in the cluster use [Nodes API Documentation].


### Stop a Job


<Tabs
defaultValue="CLI"
values={[
{label: 'CLI', value: 'CLI'},
{label: 'API', value: 'API'},
]}>
<TabItem value="CLI">

    `bacalhau cancel [id] [flags]`

    You can use the command with [appropriate flags] to cancel a job that was previously submitted and stop it running if it has not yet completed.

</TabItem>
<TabItem value="API">

    Endpoint: `DELETE /api/v1/orchestrator/jobs/:jobID`

    You can use [Stop Job API Documentation] to terminate a specific job asynchronously.

</TabItem>
</Tabs>

 
### Job History


<Tabs
defaultValue="CLI"
values={[
{label: 'CLI', value: 'CLI'},
{label: 'API', value: 'API'},
]}>
<TabItem value="CLI">

    `bacalhau job history [id] [flags]`

    You can use the command with [appropriate flags] to enumerate the historical events related to a job, identified by its ID.

</TabItem>
<TabItem value="API">

    Endpoint: `GET /api/v1/orchestrator/jobs/:jobID/history`

    You can use [Job History API Documentation] to retrieve historical events for a specific job.

</TabItem>
</Tabs>

 
### Job Logs


You can use the `bacalhau logs [flags] [id]` [command] to retrieve the log output (stdout, and stderr) from a job. If the job is still running it is possible to follow the logs after the previously generated logs are retrieved. 

:::info
 To familiarize yourself with all the commands used in Bacalhau, please view [CLI Commands] (refer to bacalhau cli version v1.0.3) and [CLI Commands (Experimental)] (refer to experimental bacalhau cli version v1.1.0.).
 :::
