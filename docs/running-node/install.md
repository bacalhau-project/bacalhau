---
sidebar_label: 'Install'
sidebar_position: 110
---

# Install

### Install bacalhau binary
So we can run `bacalhau serve` - we first need to [install the bacalhau binary](/getting-started/installation#prerequisite-install-bacalhau-client)

### Install docker
So we can run docker based workloads - we need to have [docker installed](https://docs.docker.com/engine/install/) and running.

You can configure the connection to Docker with the following environment variables:

 * `DOCKER_HOST` to set the url to the docker server.
 * `DOCKER_API_VERSION` to set the version of the API to reach, leave empty for latest.
 * `DOCKER_CERT_PATH` to load the TLS certificates from.
 * `DOCKER_TLS_VERIFY` to enable or disable TLS verification, off by default.

### Windows support
Running a Windows-based node is not officially supported but does mostly work – your mileage may vary. In particuar, some features (such as [resource limits](./resource-limits)) are not present.

Bacalhau currently assumes that all containers are Linux-based. Users of the Docker executor will need to manually ensure their Docker engine is running and [configured appropriately](https://docs.docker.com/desktop/install/windows-install/) to support Linux containers, e.g. using the WSL-based backend.
