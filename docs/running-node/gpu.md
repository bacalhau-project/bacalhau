---
sidebar_label: 'GPU Support' sidebar_position: 101
description: How to enable GPU support on your Bacalhau node.
---

# GPU Support

Bacalhau supports GPUs out of the box and defaults to allowing execution on all GPUs installed on the node.

## Prerequisites

Bacalhau assumes you have installed all the necessary NVIDIA drivers on your node. Bacalhau requires:

* [docker](https://get.docker.com/)
* [cuda-drivers for your GPU](https://docs.nvidia.com/datacenter/tesla/tesla-installation-notes/index.html)
* [NVIDIA Container Toolkit (nvidia-docker2)](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html)
* [permission to access Docker](https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user)

You can test whether you have a working GPU setup with the following command:

```bash
docker run --rm --gpus all nvidia/cuda:11.0.3-base-ubuntu20.04 nvidia-smi
```

### Unofficial GPU Node Setup One-Liner

You should review the documented links above for official install instructions, but if you're desperate for something to copy and paste, this should work:

```bash
sudo apt update
sudo apt-get install -y linux-headers-$(uname -r)
distribution=$(. /etc/os-release;echo $ID$VERSION_ID | sed -e 's/\.//g') && wget https://developer.download.nvidia.com/compute/cuda/repos/$distribution/x86_64/cuda-keyring_1.0-1_all.deb && sudo dpkg -i cuda-keyring_1.0-1_all.deb
sudo apt-get update && sudo apt-get -y install cuda-drivers
curl https://get.docker.com | sh \
  && sudo systemctl --now enable docker
distribution=$(. /etc/os-release;echo $ID$VERSION_ID) \
      && curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
      && curl -s -L https://nvidia.github.io/libnvidia-container/$distribution/libnvidia-container.list | \
            sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
            sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
distribution=$(. /etc/os-release;echo $ID$VERSION_ID) \
      && curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
      && curl -s -L https://nvidia.github.io/libnvidia-container/experimental/$distribution/libnvidia-container.list | \
         sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
         sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
sudo apt-get update
sudo apt-get install -y nvidia-docker2
sudo systemctl restart docker
sudo groupadd docker && sudo usermod -aG docker $USER && newgrp docker 
docker run --rm --gpus all nvidia/cuda:11.0.3-base-ubuntu20.04 nvidia-smi
```

This was tested on a VM in GCP, in `europe-west1-b`, of type `n1-highmem-4`, with `1 x NVIDIA Tesla T4`.

## GPU Node Configuration

The following settings refer to the `bacalhau serve` command. To see all settings please refer to the [CLI documentation](../cli-flags/all-flags.md).

### Adding Global GPU Limits

To limit the number of GPUs that Bacalhau has access to use the `limit-total-gpu` flag.

### Adding Job GPU Limits

To limit the number of GPUs that individual jobs can use use the `limit-job-gpu` flag.

## Limitations

See [GPU page in the getting started guide](../getting-started/gpu.md#limitations).