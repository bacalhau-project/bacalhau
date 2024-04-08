---
sidebar_label: approve
---

# Command: `node approve`

The `bacalhau node approve` command offers administrators the ability to approve the cluster membership for a node using its name.

## Description:

Using the `approve` sub-command under the `bacalhau node` umbrella, users can allow a node in the pending state to join the cluster and receive work. This feature is crucial for system administrators to manage the cluster.

## Usage:

```bash
bacalhau node approve [id] [flags]
```

## Flags:

- `[id]`:

  - The unique identifier of the node you wish to describe.

- `-h`, `--help`:

  - Displays the help documentation for the `describe` command.

- `-m message`:

  - A message to be attached to the approval action.

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

1. Approve a Node with ID `nodeID123`:

   ```bash
   bacalhau node approve nodeID123
   ```

2. Approve a Node with an audit message:

   ```bash
   bacalhau node approve nodeID123 -m "okay"
   ```
