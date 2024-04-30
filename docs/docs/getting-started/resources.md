---
sidebar_label: 'Hardware/GPU setup'
sidebar_position: 4
---

# Hardware Setup

Different jobs may require different amounts of resources to execute. Some jobs may have specific hardware requirements, such as GPU. This page describes how to specify hardware requirements for your job.

:::info
Please bear in mind that each executor is implemented independently and these docs might be slightly out of date. Double check the man page for the executor you are using with `bacalhau [executor] --help`.
:::

## Docker Executor

The following table describes how to specify hardware requirements for the Docker executor.


Flag | Default | Description
---------|----------|---------
 `--cpu` | 0.1 ([source](/)) | Job CPU cores (e.g. 500m, 2, 8)
 `--memory` | 100MB ([source](/)) | Job Memory requirement (e.g. 500Mb, 2Gb, 8Gb).
 `--gpu` | 0 ([source](/)) | Job GPU requirement (e.g. 1).


### How it Works

When you specify hardware requirements, the job will be offered out to the network to see if there are any nodes that can satisfy the requirements. If there are, the job will be scheduled on the node and the executor will be started.


## GPU Setup

Bacalhau supports GPU workloads. Learn how to run a job using GPU workloads with the Bacalhau client.

### Prerequisites

* The Bacalhau network must have an executor node with a GPU exposed
* Your container must include the CUDA runtime (cudart) and must be compatible with the CUDA version running on the node

## Usage

Use following command to see available resources amount:

```bash
bacalhau node list --show=capacity
```

To submit a request for a job that requires more than the standard set of resources, add the `--cpu` and `--memory` flags. For example, for a job that requires 2 CPU cores and 4Gb of RAM, use `--cpu=2 --memory=4Gb`, e.g.:

```bash
bacalhau docker run ubuntu echo Hello World --cpu=2 --memory=4Gb
```

To submit a GPU job request, use the `--gpu` flag under the `docker run` command to select the number of GPUs your job requires. For example:

```bash
bacalhau docker run --gpu=1 nvidia/cuda:11.0.3-base-ubuntu20.04 nvidia-smi
```

### Limitations

The following limitations currently exist within Bacalhau.

* Maximum CPU and memory limits depend on the participants in the network
* For GPU:
    * NVIDIA, Intel or AMD GPUs only
    * Only the Docker Executor supports GPUs
