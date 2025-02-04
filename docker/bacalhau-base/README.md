# Bacalhau Base Image

This is the standard Bacalhau container image, suitable for running orchestrator nodes, clients, and compute nodes with non-Docker execution engines (like WASM).

## Image Information

- Base Image: `ubuntu:24.04`
- Registry: `ghcr.io/bacalhau-project/bacalhau`
- Tags:
    - `latest`: Most recent stable release
    - `vX.Y.Z`: Specific version (e.g., `v1.6.0`)

## Use Cases

This image is ideal for:
- Running orchestrator nodes
- Running the Bacalhau client for job submission
- Running compute nodes that don't require Docker execution capabilities

## Usage Examples

### Running an Orchestrator Node

```bash
docker run ghcr.io/bacalhau-project/bacalhau:latest serve --orchestrator
```

### Using as a Client

```bash
docker run ghcr.io/bacalhau-project/bacalhau:latest list
```

### Running a WASM Compute Node

```bash
docker run ghcr.io/bacalhau-project/bacalhau:latest serve --compute 
```

### Running a Specific Version

```bash
docker run ghcr.io/bacalhau-project/bacalhau:v1.6.0 serve
```

## Features

- Minimal image size
- Standard Ubuntu-based environment
- Support for orchestrator nodes
- Support for client operations
- Support for WASM compute nodes
- Multi-architecture support (amd64/arm64)

## When to Use This Image

Use this image when:
- Running orchestrator nodes[README.md](../bacalhau-dind/README.md)
[README.md](README.md)
- Using Bacalhau as a client
- Running compute nodes with WASM execution
- Running in environments where Docker-in-Docker is not needed or desired
- Minimal container footprint is desired

For compute nodes requiring Docker execution capabilities, use the DinD variant instead (`bacalhau:latest-dind`).

## Additional Resources

- [Bacalhau Documentation](https://docs.bacalhau.org/)
- [GitHub Repository](https://github.com/bacalhau-project/bacalhau)
- [Getting Started Guide](https://docs.bacalhau.org/getting-started)