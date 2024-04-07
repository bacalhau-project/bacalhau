---
sidebar_label: 'GPU Support'
sidebar_position: 160
description: How to enable GPU support on your Bacalhau node.
# cspell: ignore nvidia, amd, intel, rocm, xpumanager, xpumd, xpumcli, kfd, dri, nvidia-smi, rocm-smi, xpu-smi
---

# GPU Support

Bacalhau supports GPUs out of the box and defaults to allowing execution on all GPUs installed on the node.

## Prerequisites

Bacalhau makes the assumption that you have installed all the necessary drivers and tools on your node host and have appropriately configured them for use by Docker.

In general for GPUs from any vendor, the Bacalhau client requires:

* [Docker](https://get.docker.com/)
* [Permission to access Docker](https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user)

### Nvidia

* [Drivers for your GPU](https://docs.nvidia.com/datacenter/tesla/tesla-installation-notes/index.html)
* [NVIDIA Container Toolkit (nvidia-docker2)](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html)
* `nvidia-smi` installed and functional

You can test whether you have a working GPU setup with the following command:

```bash
docker run --rm --gpus all nvidia/cuda:11.0.3-base-ubuntu20.04 nvidia-smi
```

### AMD

Bacalhau requires AMD drivers to be appropriately installed and access to the
`rocm-smi` tool.

You can test whether you have a working GPU setup with the following command,
which should print details of your GPUs:

```bash
docker run --rm --device=/dev/kfd --device=/dev/dri --entrypoint=rocm-smi rocm/rocm-terminal
```

### Intel

Bacalhau requires appropriate Intel drivers to be installed and access to the
`xpu-smi` tool.

You can test whether you have a working GPU setup with the following command,
which should print details of your GPUs:

```bash
docker run --rm --device=/dev/dri --entrypoint=/bin/bash intel/xpumanager -- -c 'xpumd & sleep 5; xpumcli discovery'
```

## GPU Node Configuration

Access to GPUs can be controlled using [resource limits](./resource-limits.md).
To limit the number of GPUs that can be used per job, set a job resource limit.
To limit access to GPUs from all jobs, set a total resource limit.
