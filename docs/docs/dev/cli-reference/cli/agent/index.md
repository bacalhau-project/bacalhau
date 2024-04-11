# command: `agent`

## Description

The `bacalhau agent` command is a parent command that offers sub-commands to query information about the Bacalhau agent. This can be useful for debugging, monitoring, or managing the agent's behavior and health.

## Usage

```
bacalhau agent [command]
```

## Available Commands

1. **[alive](./alive)**:
    - Description: Retrieves the agent's liveness and health information. This can be helpful to determine if the agent is running and healthy.
    - Usage:
        ```bash
        bacalhau agent alive
        ```

2. **[node](./node)**:
    - Description: Gathers the agent's node-related information. This might include details about the machine or environment where the agent is running, available resources, supported engines, etc.
    - Usage:
        ```bash
        bacalhau agent node
        ```

3. **[version](./version)**:
    - Description: Retrieves the Bacalhau version of the agent. This can be beneficial for ensuring compatibility or checking for updates.
    - Usage:
        ```bash
        bacalhau agent version
        ```

For more detailed information on any of the sub-commands, you can use the command:
```bash
bacalhau agent [command] --help
```

## Flags

- `-h`, `--help`:
    - Description: Displays help information for the `agent` command.


## Global Flags

- `--api-host string`:
    - Description: Specifies the host used for RESTful communication between the client and server. The flag is disregarded if the `BACALHAU_API_HOST` environment variable is set.
    - Default: `bootstrap.production.bacalhau.org`

- `--api-port int`:
    - Description: Specifies the port for REST communication. If the `BACALHAU_API_PORT` environment variable is set, this flag will be ignored.
    - Default: `1234`

- `--log-mode logging-mode`:
    - Description: Sets the desired log format. Options are: `default`, `station`, `json`, `combined`, and `event`.
    - Default: `default`

- `--repo string`:
    - Description: Defines the path to the bacalhau repository.
    - Default: ``$HOME/.bacalhau`
