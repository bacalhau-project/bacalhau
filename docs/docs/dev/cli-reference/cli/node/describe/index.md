---
sidebar_label: describe
---
# Command: `node describe`

The `bacalhau node describe` command offers users the ability to retrieve detailed information about a specific node using its unique identifier.

## Description:
Using the `describe` sub-command under the `bacalhau node` umbrella, users can get comprehensive details of a node by providing its ID. This information is crucial for system administrators and network managers to understand the state, specifications, and other attributes of nodes in their infrastructure.

## Usage:
```bash
bacalhau node describe [id] [flags]
```

## Flags:

- `[id]`:
  - The unique identifier of the node you wish to describe.

- `-h`, `--help`:
  - Displays the help documentation for the `describe` command.

- `--output format`:
  - Defines the desired format for the command's output.
  - Options: `"json"` or `"yaml"`
  - Default: `"yaml"`

- `--pretty`:
  - When this flag is used, the command will pretty print the output. This is applicable only for outputs in `json` and `yaml` formats.

## Global Flags:

- `--api-host string`:
  - Specifies the host for client-server communication through REST. This flag is overridden if the `BACALHAU_API_HOST` environment variable is set.
  - Default: `"bootstrap.production.bacalhau.org"`

- `--api-port int`:
  - Designates the port for REST-based communication between client and server. This flag is overlooked if the `BACALHAU_API_PORT` environment variable is defined.
  - Default: `1234`

- `--log-mode logging-mode`:
  - Determines the log format preference.
  - Options: `'default','station','json','combined','event'`
  - Default: `'default'`

- `--repo string`:
  - Points to the bacalhau repository's path.
  - Default: `"`$HOME/.bacalhau"`

## Examples:

1. Describing a Node with ID `nodeID123`:
   ```bash
   bacalhau node describe nodeID123
   ```

2. Describing a Node with Output in JSON Format:
   ```bash
   bacalhau node describe nodeID123 --output json
   ```

3. Pretty Printing the Description of a Node:
   ```bash
   bacalhau node describe nodeID123 --pretty
   ```
