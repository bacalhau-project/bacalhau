---
sidebar_label: 'Create Private Network'
sidebar_position: 5
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# Create Private Network

## Introduction
Bacalhau allows you to create your own private network so you can securely run private workloads without the risks inherent in working on public nodes or inadvertently distributing data outside your organization.

This tutorial describes the process of creating your own private network from multiple nodes, configuring the nodes, and running demo jobs.

## TL;DR

1. [Install Bacalhau](../getting-started/installation.md) `curl -sL https://get.bacalhau.org/install.sh | bash` on every host
1. Start the [Requester node](#initial-requester-node): `bacalhau serve --node-type requester`
1. Copy and paste the command it outputs under the "*To connect a compute node to this orchestrator, run the following command in your shell*" line to **other hosts**
1. Copy and paste the environment variables it outputs under the "*To connect to this node from the client, run the following commands in your shell*" line to a **client machine**
1. Done! Run sample hello-world command on the client machine `bacalhau docker run apline echo hello`


## Prerequisites

1. Prepare the hosts on which the nodes are going to be set up. They could be:
    1. Physical Hosts
    1. Cloud VMs ([AWS](https://aws.amazon.com/ec2/), [GCP](https://cloud.google.com/products/compute), [Azure](https://azure.microsoft.com/en-us/products/virtual-machines) or any other provider)
    1. Local Hypervisor VMs
    1. [Docker Containers](../setting-up/running-node/quick-start-docker.md)
1. [Install Bacalhau](../getting-started/installation.md) on each host
1. Ensure that all nodes are connected to the same network and that the necessary ports are open for communication between them.
   1. Ensure your nodes have an internet connection in case you have to download or upload any data (docker images, input data, results)
1. Ensure that [Docker Engine](https://docs.docker.com/engine/install/) is installed in case you are going to run Docker Workloads

:::info
Bacalhau is designed to be versatile in its deployment, capable of running on various environments: physical hosts, virtual machines or cloud instances. Its resource requirements are modest, ensuring compatibility with a wide range of hardware configurations. However, for certain workloads, such as machine learning, it's advisable to consider hardware configurations optimized for computational tasks, including [GPUs](../setting-up/running-node/gpu).
:::

## Start Initial Requester Node

The Bacalhau network consists of nodes of two types: compute and requester. Compute Node is responsible for executing jobs and producing results. Requester Node is responsible for handling user requests, forwarding jobs to compute nodes and monitoring the job lifecycle.

The first step is to start up the initial **Requester** node. This node will connect to nothing but will listen for connections. 

Start by creating a secure token. This token will be used for authentication between the orchestrator and compute nodes during their communications. Any string can be used as a token, preferably not easy to guess or bruteforce. In addition, new authentication methods will be introduced in future releases. 

### Create and Set Up a Token

Let's use the `uuidgen` tool to create our token, then add it to the Bacalhau configuration and run the requester node:

```bash
# Create token and write it into the 'my_token' file
uuidgen > my_token

#Add token to the Bacalhau configuration
bacalhau config set "node.network.authsecret" my_token
```
```bash
#Start the Requester node
bacalhau serve --node-type requester --peer none
```

This will produce output similar to this, indicating that the node is up and running:

```
15:09:58.711 | INF pkg/nats/logger.go:47 > Starting nats-server [Server:n-1134cdf3-a974-4c0b-b9c9-61858a856bda]
...
15:09:58.719 | INF pkg/nats/logger.go:47 > Server is ready [Server:n-1134cdf3-a974-4c0b-b9c9-61858a856bda]
15:09:58.739 | INF pkg/nats/server.go:48 > NATS server NAN464RLFLYVA7GYZ6QN3RSH6UAKFJHMQWON4K4VVIRE3O3C7RKU3V7D listening on nats://0.0.0.0:4222 [NodeID:n-1134cdf3]
15:10:02.81 | INF pkg/config/setters.go:84 > Writing to config file /home/username/.bacalhau/config.yaml:
Node.Compute.ExecutionStore:    {BoltDB /home/username/.bacalhau/compute_store/executions.db}
Node.Requester.JobStore:        {BoltDB /home/username/.bacalhau/orchestrator_store/jobs.db}
Node.Name:      n-1134cdf3-a974-4c0b-b9c9-61858a856bda

To connect a compute node to this orchestrator, run the following command in your shell:
bacalhau serve --node-type=compute --network=nats --orchestrators=nats://127.0.0.1:4222 --private-internal-ipfs --ipfs-swarm-addrs=/ip4/127.0.0.1/tcp/39311/p2p/QmdUmWyEUHK3Zfnno4x3Ct89AjtQ75Tr3WGaEsgh1nGGj1 

To connect to this node from the client, run the following commands in your shell:
export BACALHAU_NODE_CLIENTAPI_HOST=0.0.0.0
export BACALHAU_NODE_CLIENTAPI_PORT=1234
export BACALHAU_NODE_NETWORK_TYPE=nats
export BACALHAU_NODE_NETWORK_ORCHESTRATORS=nats://127.0.0.1:4222
export BACALHAU_NODE_IPFS_SWARMADDRESSES=/ip4/127.0.0.1/tcp/39311/p2p/QmdUmWyEUHK3Zfnno4x3Ct89AjtQ75Tr3WGaEsgh1nGGj1
```

Note that for security reasons, the output of the command contains the localhost `127.0.0.1` address instead of your real IP. To connect to this node, you should replace it with your real public IP address yourself. The method for obtaining your public IP address may vary depending on the type of instance you're using. Windows and Linux instances can be queried for their public IP using the following command:
```bash
curl https://api.ipify.org
```
If you are using a cloud deployment, you can find your public IP through their console, e.g. [AWS](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-instance-addressing.html#:~:text=during%20instance%20launch-,View%20the%20IPv4%20addresses,-You%20can%20use) and [Google Cloud](https://cloud.google.com/compute/docs/instances/view-ip-address#:~:text=Viewing%20IP%20addresses,-You%20view%20the&text=In%20the%20Google%20Cloud%20console,address%2C%20you%20can%20assign%20one.)

## Create and Connect Compute Node

Now let's move to another host from the preconditions, start a compute node on it and connect to the requester node. Here you will also need to add the same token to the configuration as on the requester.

```bash
#Add token to the Bacalhau configuration
bacalhau config set "node.network.authsecret" my_token
```
Then execute the `serve` command to connect to the requester node: 
```
bacalhau serve --node-type=compute --orchestrators=<Public-IP-of-Requester-Node>
```
This will produce output similar to this, indicating that the node is up and running:

```bash
15:51:02.534 | INF pkg/publisher/local/server.go:52 > Running local publishing server on 0.0.0.0:6001 [NodeID:n-ef98aa76]

To connect to this node from the client, run the following commands in your shell:
export BACALHAU_NODE_CLIENTAPI_HOST=0.0.0.0
export BACALHAU_NODE_CLIENTAPI_PORT=1234

A copy of these variables have been written to: /home/username/.bacalhau/bacalhau.run
```

To ensure that the nodes are connected to the network, run the following command, specifying the public IP of the requester node:

```bash
bacalhau --api-host <Public-IP-of-Requester-Node> node list
```
This will produce output similar to this, indicating that the nodes belong to the same network:
```bash
bacalhau --api-host 10.0.2.15 node list
 ID          TYPE       STATUS    LABELS                                              CPU     MEMORY      DISK         GPU  
 n-550ee0db  Compute              Architecture=amd64 Operating-System=linux           0.8 /   1.5 GB /    12.3 GB /    0 /  
                                  git-lfs=true                                        0.8     1.5 GB      12.3 GB      0    
 n-b2ab8483  Requester  APPROVED  Architecture=amd64 Operating-System=linux                                                 
```


## Submitting Jobs

To connect to the requester node find the following lines in the requester node logs:

```bash
To connect to this node from the client, run the following commands in your shell:
export BACALHAU_NODE_CLIENTAPI_HOST=<Public-IP-of-the-Requester-Node>
export BACALHAU_NODE_CLIENTAPI_PORT=1234
export BACALHAU_NODE_NETWORK_TYPE=nats
export BACALHAU_NODE_NETWORK_ORCHESTRATORS=nats://<Public-IP-of-the-Requester-Node>:4222
export BACALHAU_NODE_IPFS_SWARMADDRESSES=/ip4/<Public-IP-of-the-Requester-Node>/tcp/43919/p2p/QmehkJQ9BN4QMvv7nFTzsWSBk13coaxEZh4N5YmumtJQDb
```

:::info
The exact commands list will be different for each node and is outputted by the `bacalhau serve` command.
:::

:::info
Note that by default such command contains `127.0.0.1` or `0.0.0.0` instead of actual public IP. Make sure to replace it before executing the command.
:::

Now you can submit your jobs using the `bacalhau docker run`, `bacalhau wasm run` and `bacalhau job run` commands. For example submit a hello-world job `bacalhau docker run alpine echo hello`:

```bash
bacalhau docker run alpine echo hello

Using default tag: latest. Please specify a tag/digest for better reproducibility. 
Job successfully submitted. Job ID: ddbfa358-d663-4f54-804e-598c53dbb969

Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):

        Communicating with the network  ................  done ✅  0.0s
           Creating job for submission  ................  done ✅  0.5s
                       Job in progress  ................  done ✅  0.0s

To download the results, execute:
        bacalhau get ddbfa358-d663-4f54-804e-598c53dbb969
        
To get more details about the run, execute: 
        bacalhau describe ddbfa358-d663-4f54-804e-598c53dbb969 
```

You will be able to see the job execution logs on the compute node:

```bash
15:42:06.32 | INF pkg/executor/docker/executor.go:116 > starting execution [NodeID:n-550ee0db] [execution:e-f79b74aa-82c3-4fbe-ac71-476f0d596161] [executionID:e-f79b74aa-82c3-4fbe-ac71-476f0d596161] [job:ddbfa358-d663-4f54-804e-598c53dbb969] [jobID:ddbfa358-d663-4f54-804e-598c53dbb969]

...

15:42:06.665 | INF pkg/executor/docker/executor.go:217 > received results from execution [executionID:e-f79b74aa-82c3-4fbe-ac71-476f0d596161]
15:42:06.676 | INF pkg/compute/executor.go:195 > cleaning up execution [NodeID:n-550ee0db] [execution:e-f79b74aa-82c3-4fbe-ac71-476f0d596161] [job:ddbfa358-d663-4f54-804e-598c53dbb969]
```

## Publishers and Sources Configuration

By default, IPFS & Local publishers and URL & IPFS sources are available on the compute node. The following describes how to configure the appropriate sources and publishers:

<Tabs
defaultValue="S3"
values={[
{label: 'S3', value: 'S3'},
{label: 'IPFS', value: 'IPFS'},
{label: 'Local', value: 'Local'},
]}>
<TabItem value="S3">

To set up [S3 publisher](../setting-up/other-specifications/publishers/s3.md) you need to specify environment variables such as `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`, populating a credentials file to be located on your compute node, i.e. `~/.aws/credentials`, or creating an [IAM role](https://aws.amazon.com/iam/) for your compute nodes if you are utilizing cloud instances.

Your chosen publisher can be set for your Bacalhau compute nodes declaratively or imperatively using either configuration yaml file:
```yaml
Publisher:
  Type: "s3"
  Params:
    Bucket: "my-task-results"
    Key: "task123/result.tar.gz"
    Endpoint: "https://s3.us-west-2.amazonaws.com"
```
Or within your imperative job execution commands:
```bash
bacalhau docker run -p s3://bucket/key,opt=endpoint=http://s3.example.com,opt=region=us-east-1 ubuntu …
```
S3 compatible publishers can also be used as [input sources](../setting-up/other-specifications/sources/s3.md) for your jobs, with a similar configuration.
```yaml
InputSources:
  - Source:
      Type: "s3"
      Params:
        Bucket: "my-bucket"
        Key: "data/"
        Endpoint: "https://storage.googleapis.com"
  - Target: "/data"
```
</TabItem>
<TabItem value="IPFS">
By default, bacalhau creates its own in-process IPFS node that will attempt to discover other IPFS nodes, including public nodes, on its own.
If you specify the `--private-internal-ipfs` flag when starting the node, the node will not attempt to discover other nodes. Note, that such an IPFS node exists only with the compute node and will be shut down along with it.
Alternatively, you can create your own private IPFS network and connect to it using the [appropriate flags](../dev/cli-reference/all-flags.md#serve). 

[IPFS publisher](../setting-up/other-specifications/publishers/ipfs.md) can be set for your Bacalhau compute nodes declaratively or imperatively using either configuration yaml file:

```yaml
Publisher:
  Type: ipfs
```

Or within your imperative job execution commands:

```bash
bacalhau docker run --publisher ipfs ubuntu ...
```

Data pinned to the IPFS network can be used as [input source](../setting-up/other-specifications/sources/ipfs.md). To do this, you will need to specify the CID in declarative:
```yaml
InputSources:
  - Source:
      Type: "ipfs"
      Params:
        CID: "QmY7Yh4UquoXHLPFo2XbhXkhBvFoPwmQUSa92pxnxjY3fZ"
  - Target: "/data"
```

Or imperative format:
```bash
bacalhau docker run --input QmY7Yh4UquoXHLPFo2XbhXkhBvFoPwmQUSa92pxnxjY3fZ:/data ...
```
</TabItem>
<TabItem value="Local">

Bacalhau allows to publish job results directly to the compute node. Please note that this method is not a reliable storage option and is recommended to be used mainly for introductory purposes.

[Local publisher](../setting-up/other-specifications/publishers/local.md) can be set for your Bacalhau compute nodes declaratively or imperatively using configuration yaml file:

```yaml
Publisher:
  Type: local
```

Or within your imperative job execution commands:

```bash
bacalhau docker run --publisher local ubuntu ...
```

The [Local input source](../setting-up/other-specifications/sources/local.md) allows Bacalhau jobs to access files and directories that are already present on the compute node. To allow jobs to access local files when starting a node, the `--allow-listed-local-paths` flag should be used, specifying the path to the data and access mode `:rw` for Read-Write access or `:ro` for Read-Only (used by default). For example:

```bash
bacalhau serve --allow-listed-local-paths "/etc/config:rw,/etc/*.conf:ro"
```

Further, the path to local data in declarative or imperative form must be specified in the job. Declarative example of the local input source:
```yaml
InputSources:
  - Source:
      Type: "localDirectory"
      Params:
        SourcePath: "/etc/config"
        ReadWrite: true
    Target: "/config"
```
Imperative example of the local input source:

```bash
bacalhau docker run -input file:///etc/config:/config ubuntu ...
```

</TabItem>
</Tabs>


## Best Practices for Production Use Cases
Your private cluster can be quickly set up for testing packaged jobs and tweaking data processing pipelines. However, when using a private cluster in production, here are a few considerations to note.

1. Ensure you are running the Bacalhau process from a dedicated system user with limited permissions. This enhances security and reduces the risk of unauthorized access to critical system resources. If you are using an orchestrator such as [Terraform](https://www.terraform.io/), utilize a service file to manage the Bacalhau process, ensuring the correct user is specified and consistently used. Here’s a [sample service file](https://github.com/bacalhau-project/bacalhau/blob/main/ops/marketplace-tf/modules/instance_files/bacalhau.service)
1. Create an authentication file for your clients. A [dedicated authentication file or policy](../dev/auth_flow.md) can ease the process of maintaining secure data transmission within your network. With this, clients can authenticate themselves, and you can limit the Bacalhau API endpoints unauthorized users have access to.
1. Consistency is a key consideration when deploying decentralized tools such as Bacalhau. You can use an [installation script](https://github.com/bacalhau-project/bacalhau/blob/main/ops/marketplace-tf/modules/instance_files/install-bacalhau.sh#L5) to affix a specific version of Bacalhau or specify deployment actions, ensuring that each host instance has all the necessary resources for efficient operations.
1. Ensure separation of concerns in your cloud deployments by mounting the Bacalhau repository on a separate non-boot disk. This prevents instability on shutdown or restarts and improves performance within your host instances.


That's all folks! 🎉 Please contact us on [Slack](https://bacalhauproject.slack.com) `#bacalhau` channel for questions and feedback!