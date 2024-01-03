---
sidebar_label: node
---
# Command: `agent node`

## Description

The `bacalhau agent node` command retrieves information about the agent's node, providing insights into the agent's environment and aiding in debugging.

## Usage

```bash
bacalhau agent node [flags]
```

## Flags

- `-h`, `--help`:
  - Displays help information for the `node` sub-command.

- `--output format`:
  - Defines the output format (either JSON or YAML).
  - Options: `json`, `yaml`
  - Default: `yaml`

- `--pretty`:
  - Beautifies the output when using JSON or YAML formats.

## Global Flags

- `--api-host string`:
  - The host for REST communication. Overrides the `BACALHAU_API_HOST` environment variable.
  - Default: `bootstrap.production.bacalhau.org`

- `--api-port int`:
  - The port for REST communication. Overridden if `BACALHAU_API_PORT` environment variable is set.
  - Default: `1234`

- `--log-mode logging-mode`:
  - Specifies the log format. Choices are: `default`, `station`, `json`, `combined`, `event`.
  - Default: `default`

- `--repo string`:
  - Path to the bacalhau repository.
  - Default: ``$HOME/.bacalhau`

## Examples

1. **Retrieve Node Information in Default Format (YAML)**

   ```bash
   bacalhau agent node
   ```

2. **Retrieve Node Information in JSON Format**

   ```bash
   bacalhau agent node --output json
   ```

3. **Retrieve Node Information in Pretty-printed JSON Format**

   ```bash
   bacalhau agent node --output json --pretty
   ```
