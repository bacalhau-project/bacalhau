# Running Bacalhau on Docker

## Overview

Since Bacalhau is a distributed system with multiple components, it is critical to have a reliable method for end-to-end testing. Additionally, it's important that these tests closely resemble a real production environment without relying on mocks.

This setup addresses those needs by running Bacalhau inside containers while also supporting Docker workloads within these containers (using Docker-in-Docker, or DinD).

## Architecture

- A Requester Docker container, running Bacalhau as a requester node.
- A Compute Docker container, running Bacalhau as a compute node and is configured to run Docker containers inside it.
- A Bacalhau Client Docker container to act as a jumpbox to interact with this Bacalhau deployment.
- A [Registry](https://github.com/distribution/distribution/) Docker container to act as the local container image registry.
- A Minio Docker container to support running S3 compatible input/output jobs.
- Docker Compose is used to create 5 services: the Requester Node, the Compute Node, the Client CLI Node, the registry node, and the Minio node.
- All the services are connected on the same Docker network, allowing them to communicate over the bridged network.
- All the containers have an injected custom Certificate Authority, which is used for a portion of the internal TLS communication.
  - TODO: Expand the TLS setup to more components. Now it is used for the registry communication only.

## Setup

---
### Build the Docker Images

Build the Requester Node image:
```shell
docker build -f Dockerfile-RequesterNode -t bacalhau-requester-node-image .
```

Build the Compute Node image:
```shell
docker build -f Dockerfile-ComputeNode -t bacalhau-compute-node-image .
```

Build the Client Node image:
```shell
docker build -f Dockerfile-ClientNode -t bacalhau-client-node-image .
```

Build the Registry Node image:
```shell
docker build -f Dockerfile-DockerImageRegistryNode -t bacalhau-container-img-registry-node-image .
```

After running these commands, you should see the above images created:
```shell
docker image ls
```
---
### Running the setup

Run Docker Compose:
```shell
docker-compose up
```

Access the utility client container to use the Bacalhau CLI:
```shell
docker exec -it bacalhau-client-node-container /bin/bash
```

Once inside the container, you can run the following commands to verify the setup:
```shell
# You should see two nodes: a Requestor and a Compute Node
bacalhau node list
```

Run a test workload
```shell
bacalhau docker run hello-world

# Describe the job; it should have completed successfully.
bacalhau job describe ........
```

In another terminal window, you can follow the logs of the Requester node, and compute node
```shell
docker logs bacalhau-requester-node-container -f
docker logs bacalhau-compute-node-container -f
```

---
### Setting Up Minio

Access the utility client container to use the Bacalhau CLI:
```shell
docker exec -it bacalhau-client-node-container /bin/bash
```

Setup an alias for the Minio CLI
```shell
# The environment variables are already injected in
# the container, no need to replace them yourself.
mc alias set bacalhau-minio "http://${BACALHAU_MINIO_NODE_HOST}:9000" "${MINIO_ROOT_USER}" "${MINIO_ROOT_PASSWORD}"
mc admin info bacalhau-minio
```

Create a bucket and add some files
```shell
mc mb bacalhau-minio/my-data-bucket
mc ls bacalhau-minio/my-data-bucket/section1/
echo "This is a sample text hello hello." > example.txt
mc cp example.txt bacalhau-minio/my-data-bucket/section1/
```

RUn a job with data input from the minion bucket

```shell
# Content of aws-test-job.yaml below
bacalhau job run aws-test-job.yaml
```

```yaml
Name: S3 Job Data Access Test
Type: batch
Count: 1
Tasks:
  - Name: main
    Engine:
      Type: docker
      Params:
        Image: ubuntu:latest
        Entrypoint:
          - /bin/bash
        Parameters:
          - "-c"
          - "cat /put-my-s3-data-here/example.txt"
    InputSources:
      - Target: "/put-my-s3-data-here"
        Source:
          Type: s3
          Params:
            Bucket: "my-data-bucket"
            Key: "section1/"
            Endpoint: "http://bacalhau-minio-node:9000"
            Region: "us-east-1" # If no region added, it fails, even for minio
```

---
### Setting Up private registry

This docker compose deployment has a private registry deployed on its own node. It allows us to
create tests and experiment with docker images jobs without the need to use DockerHub in anyway.

From inside the client container, let's pull an image from DockerHub, push it to our own private registry,
then run a docker job running the image in out private registry.

```shell
# pull from docker hub
docker pull ubuntu

# tag the image to prepare it to be push to our private registry
docker image tag ubuntu bacalhau-container-img-registry-node:5000/firstbacalhauimage

# push the image to our private registry
docker push bacalhau-container-img-registry-node:5000/firstbacalhauimage
```

Now, let's create a job that references that image in private registry:

```shell
# Content of private-registry-test-job.yaml below
bacalhau job run private-registry-test-job.yaml
```

```yaml
Name: Job to test using local registry images
Type: batch
Count: 1
Tasks:
  - Name: main
    Engine:
      Type: docker
      Params:
        Image: bacalhau-container-img-registry-node:5000/firstbacalhauimage
        Entrypoint:
          - /bin/bash
        Parameters:
          - "-c"
          - "echo test-local-registry"
```

---
### Notes:

If for some reason after running `docker-compose up`, you faced issues with the Image registry node starting, try to remove the image registry docker volume by running:

```shell
# Destroy the deployment
docker-compose down

# Remove registry volume
docker volume rm test-integration_registry-volume

# Create deployment again
docker-compose up
```
