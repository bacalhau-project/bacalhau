---
sidebar_label: 'Specifying Hardware Requirements'
sidebar_position: 20
---

# Specifying Hardware Requirements

Not all jobs are created equal. Some jobs require more resources than others or have specific hardware requirements like GPUs. This page describes how to specify hardware requirements for your job.

Please bear in mind that each executor is implemented independently and these docs might be slightly out of date. Double check the man page for the executor you are using with `bacalhau [executor] --help`.

## Docker Executor

The following table describes how to specify hardware requirements for the Docker executor.


Flag | Default | Description
---------|----------|---------
 `--cpu` | 0.1 ([source](https://github.com/bacalhau-project/bacalhau/blob/main/pkg/capacitymanager/capacitymanager.go#L9)) | Job CPU cores (e.g. 500m, 2, 8)
 `--memory` | 100MB ([source](https://github.com/bacalhau-project/bacalhau/blob/main/pkg/capacitymanager/capacitymanager.go#L10)) | Job Memory requirement (e.g. 500Mb, 2Gb, 8Gb).
 `--gpu` | 0 ([source](https://github.com/bacalhau-project/bacalhau/blob/main/pkg/capacitymanager/capacitymanager.go#L11)) | Job GPU requirement (e.g. 1).

### How it Works

When you specify hardware requirements, the job will be offered out to the network to see if there are any nodes that can satisfy the requirements. If there are, the job will be scheduled on the node and the executor will be started.

If there are no nodes that can satisfy the requirements, the job will wait for a node to become available, until it times out [after 3 minutes](https://github.com/bacalhau-project/bacalhau/blob/main/pkg/computenode/config.go#L12).

### Limitations

The following limitations currently exist within Bacalhau.

* Maximum CPU and memory limits depend on the participants in the network
* See [the dedicated page](docs/next-steps/gpu.md) on GPUs to see GPU limiations

### Also See

* [GPU workload tutorial](docs/next-steps/gpu.md)
