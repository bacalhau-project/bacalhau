---
sidebar_position: 4
---

# GPU Workloads

Bacalhau supports GPU workloads.

## Prerequisites

* The Bacalhau network must have an executor node with a GPU exposed
* Your container must include the CUDA runtime (cudart) and must be compatible with the CUDA version running on the node

## Usage

To submit a job request use the `--gpu` flag under the `docker run` command to select the number of GPUs your job requires. For example:

```bash
bacalhau docker run --gpu=1 nvidia/cuda:11.0.3-base-ubuntu20.04 nvidia-smi
```

## Limitations

The following limitations currently exist within Bacalhau. Bacalhau supports:

* NVIDIA GPUs only
* a single GPU only
* the Docker executor only