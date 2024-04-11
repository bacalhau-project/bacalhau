---
sidebar_label: alive
---
# Command: `agent alive`

## Description

The `bacalhau agent alive` command provides information about the agent's liveness and health. This is essential for monitoring and ensuring that the agent is active and functioning correctly.

## Usage

```
bacalhau agent alive [flags]
```

## Flags

- `-h`, `--help`:
    - Description: Displays help information for the `alive` sub-command.

- `--output format`:
    - Description: Determines the format in which the output is displayed. Available formats include JSON and YAML.
    - Options: `json`, `yaml`
    - Default: `yaml`

- `--pretty`:
    - Description: Formats the output for enhanced readability. This flag is relevant only when using JSON or YAML output formats.

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


## Examples

### Checking the Agent's Liveness and Health Info

1. **Basic Usage**:

   **Command**:
   ```
   → bacalhau agent alive
   ```

   **Output**:
   ```
   status: OK
   ```

2. **Output in JSON format**:

   **Command**:
   ```
   → bacalhau agent alive --output json --pretty
   ```

   **Output**:
   ```json
   {
     "Status": "OK"
   }
   ```
