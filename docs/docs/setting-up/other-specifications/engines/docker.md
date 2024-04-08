---
sidebar_label: Docker
# cspell: ignore myvalue
---

# Docker Engine Specification

Docker Engine is one of the execution engines supported in Bacalhau. It allows users to run tasks inside Docker containers, offering an isolated and consistent environment for execution. Below are the parameters to configure the Docker Engine.

## `Docker` Engine Parameters

- **Image** `(string: <required>)`: Specifies the Docker image to use for task execution. It should be an image that can be pulled by Docker.

- **Entrypoint** `(string[]: <optional>)`: Allows overriding the default entrypoint set in the Docker image. Each string in the array represents a segment of the entrypoint command.

- **Parameters** `(string[]: <optional>)`: Additional command-line arguments to be included in the container’s startup command, appended after the entrypoint.

- **EnvironmentVariables** `(string[]: <optional>)`: Sets environment variables within the Docker container during task execution. Each string should be formatted as `KEY=value`.

- **WorkingDirectory** `(string: <optional>)`: Sets the path inside the container where the task executes. If not specified, it defaults to the working directory defined in the Docker image.

### Example

Here’s an example of configuring the Docker Engine within a job or task using YAML:

```yaml
Engine:
  Type: "Docker"
  Params:
    Image: "ubuntu:20.04"
    Entrypoint:
      - "/bin/bash"
      - "-c"
    Parameters:
      - "echo Hello, World!"
    EnvironmentVariables:
      - "MY_ENV_VAR=myvalue"
    WorkingDirectory: "/app"
```

In this example, the task will be executed inside an Ubuntu 20.04 Docker container. The entrypoint is overridden to execute a bash shell that runs an echo command. An environment variable MY_ENV_VAR is set with the value myvalue, and the working directory inside the container is set to /app.
