---
sidebar_label: 'GPU Installation'
sidebar_position: 160
description: How to enable GPU support on your Bacalhau node.
# cspell: ignore nvidia, amd, intel, rocm, xpumanager, xpumd, xpumcli, kfd, dri, nvidia-smi, rocm-smi, xpu-smi
---

# GPU Installation

Bacalhau supports GPUs out of the box and defaults to allowing execution on all GPUs installed on the node.

## Prerequisites

Bacalhau makes the assumption that you have installed all the necessary drivers and tools on your node host and have appropriately configured them for use by Docker.

In general for GPUs from any vendor, the Bacalhau client requires:

1. [Docker](https://docs.docker.com/engine/install/)
1. [Permission to access Docker](https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user)

### Nvidia

1. [NVIDIA GPU Drivers](https://docs.nvidia.com/datacenter/tesla/tesla-installation-notes/index.html)
2. [NVIDIA Container Toolkit (nvidia-docker2)](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html)
3. Verify installation by [Running a Sample Workload](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/sample-workload.html)
4. `nvidia-smi` installed and functional


### AMD

1. [AMD GPU drivers](https://www.amd.com/en/support/download/drivers.html)
1. `rocm-smi` tool installed and functional

See the [Running ROCm Docker containers](https://rocm.docs.amd.com/projects/install-on-linux/en/latest/how-to/docker.html) for guidance on how to run Docker workloads on AMD GPU.


### Intel

1. [Intel GPU drivers](https://www.intel.com/content/www/us/en/download-center/home.html)
1. `xpu-smi` tool installed and functional

See the [Running on GPU under docker](https://github.com/Intel-Media-SDK/MediaSDK/wiki/Running-on-GPU-under-docker) for guidance on how to run Docker workloads on Intel GPU.


## GPU Node Configuration

Access to GPUs can be controlled using [resource limits](./resource-limits.md).
To limit the number of GPUs that can be used per job, set a job resource limit.
To limit access to GPUs from all jobs, set a total resource limit.
