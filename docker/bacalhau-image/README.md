# Bacalhau Docker Image

This directory contains a Dockerfile script to build a Docker image containing the Bacalhau binary.

## Usage

You can find detailed information about the Bacalhau command on the [Bacalhau documentation website](https://docs.bacalhau.org/).

### Development Node

You can run a small Bacalhau cluster by running the `devstack` subcommand. This runs an in-memory IPFS node and several local Bacalhau nodes. This is useful for testing and development. See the [the `docs/running_locally.md` documentation](../../docs/running_locally.md) for more information.

```
docker run --env IGNORE_PID_AND_PORT_FILES=true --env PREDICTABLE_API_PORT=1 --publish 20000:20000 ghcr.io/bacalhau-project/bacalhau:latest devstack
```

### Production Node

Follow the instructions on the [Bacalhau documentation website](https://docs.bacalhau.org/running-node/quick-start) to run a Bacalhau node in production.
