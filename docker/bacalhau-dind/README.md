# Bacalhau DinD Image

This is the Docker-in-Docker (DinD) variant of the Bacalhau container image, specifically designed for running compute nodes that need to execute Docker workloads.

## Image Information
- Base Image: `docker:dind`
- Registry: `ghcr.io/bacalhau-project/bacalhau`
- Tags:
    - `latest-dind`: Most recent stable release with DinD support
    - `vX.Y.Z-dind`: Specific version (e.g., `v1.6.0-dind`)

## ⚠️ Important: Privileged Mode Required

This image MUST be run with the `--privileged` flag due to the Docker-in-Docker functionality:

```bash
docker run --privileged ghcr.io/bacalhau-project/bacalhau:latest-dind serve --compute
```

## Use Cases

This image is specifically designed for:
- Running compute nodes that execute Docker workloads
- Supporting the full range of Docker-based job execution
- Development environments requiring Docker support

## Usage Examples

### Running a Compute Node
```bash
docker run --privileged \
  ghcr.io/bacalhau-project/bacalhau:latest-dind serve --compute
```

### Development Environment
```bash
docker run --privileged \
  ghcr.io/bacalhau-project/bacalhau:latest-dind devstack
```

### Running a Specific Version
```bash
docker run --privileged \
  ghcr.io/bacalhau-project/bacalhau:v1.6.0-dind serve --compute
```

## Features
- Full Docker-in-Docker support
- Built-in Docker daemon
- Multi-architecture support (amd64/arm64)
- Automatic Docker daemon initialization

## When to Use This Image vs Base

Use this image when you need:
- Compute nodes that run Docker workloads
- Development environments with Docker capabilities
- Full container execution support

Use the base image (`bacalhau:latest`) for:
- Client operations
- Orchestrator nodes
- Compute nodes without Docker requirements
- Environments where privileged mode isn't allowed

## Troubleshooting

If you see this error:
```
ERROR: This container must be run with --privileged flag
```
Add the `--privileged` flag to your docker run command.

## Additional Resources
- [Bacalhau Documentation](https://docs.bacalhau.org/)
- [GitHub Repository](https://github.com/bacalhau-project/bacalhau)
- [Docker-in-Docker Documentation](https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities)