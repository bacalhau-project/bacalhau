# Running Bacalhau on Docker

## Overview

Since Bacalhau is a distributed system with multiple components, it is critical to have a reliable method for end-to-end testing. Additionally, it's important that these tests closely resemble a real production environment without relying on mocks.

This setup addresses those needs by running Bacalhau inside containers while also supporting Docker workloads within these containers (using Docker-in-Docker, or DinD).

## Architecture

- A Requestor Docker image is built, containing the Bacalhau CLI.
- A Compute Docker image is built with Bacalhau CLI and is configured to run Docker containers inside it.
- Docker Compose is used to create three services: the Requestor Node, the Compute Node, and the Client CLI Node.
- All three services are connected on the same Docker network, allowing them to communicate over the bridged network.

## Setup

### Build the Docker Images

Build the Compute Node image:
```shell
docker build -f Dockerfile-ComputeNode -t bacalhau-compute-node-image .
```


Build the Requestor Node image:
```shell
docker build -t bacalhau-in-docker .
```

After running these commands, you should see the two images created:
```shell
docker image ls
```

### Running the setup

Run Docker Compose:
```shell
docker-compose up
```

Access the utility container to use the Bacalhau CLI:
```shell
docker exec -it bacalhau-client-node-container /bin/bash
```

Once inside the container, you can run the following commands to verify the setup:
```shell
# You should see two nodes: a Requestor and a Compute Node
bacalhau --api-host=bacalhau-requester-node --api-port=1234 node list
```

```shell
# Run a test workload
bacalhau --api-host=bacalhau-requester-node --api-port=1234 docker run alpine echo hellooooo

# Describe the job; it should have completed successfully.
```
