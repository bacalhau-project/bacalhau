---
sidebar_label: 'Install'
sidebar_position: 110
---

# Install
This tutorial shares installation options for the Bacalhau client.

### Install the bacalhau binary
First, you should [install the bacalhau binary](/getting-started/installation#prerequisite-install-bacalhau-client) to run `bacalhau serve`. 

### Install docker
To run docker based workloads, you should have [docker installed](https://docs.docker.com/engine/install/) and running.

You can configure the connection to Docker with the following environment variables:

 * `DOCKER_HOST` to set the url to the docker server.
 * `DOCKER_API_VERSION` to set the version of the API to reach, leave empty for latest.
 * `DOCKER_CERT_PATH` to load the TLS certificates from.
 * `DOCKER_TLS_VERIFY` to enable or disable TLS verification, off by default.

### Windows support
Running a Windows-based node is not officially supported, so your mileage may vary. Some features (like [resource limits](./resource-limits)) are not present in Windows-based nodes.

Bacalhau currently makes the assumption that all containers are Linux-based. Users of the Docker executor will need to manually ensure that their Docker engine is running and [configured appropriately](https://docs.docker.com/desktop/install/windows-install/) to support Linux containers, e.g. using the WSL-based backend.
