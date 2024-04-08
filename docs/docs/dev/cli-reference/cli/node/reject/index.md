---
sidebar_label: reject
---

# Command: `node reject`

The `bacalhau node reject` command offers administrators the ability to reject a compute node's request to join the cluster.

## Description:

Using the `reject` sub-command, administrators can reject a node in the pending state from joining the cluster and receiving work. This feature is crucial for system administrators to manage the cluster and will stop the node from taking part in the cluster until approved.

## Usage:

```bash
bacalhau node rejected [id] [flags]
```

## Flags:

- `[id]`:

  - The unique identifier of the node you wish to describe.

- `-h`, `--help`:

  - Displays the help documentation for the `describe` command.

- `-m message`:

  - A message to be attached to the rejection action.

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

1. Reject a Node with ID `nodeID123`:

   ```bash
   bacalhau node reject nodeID123
   ```

2. Reject a Node with an audit message:

   ```bash
   bacalhau node reject nodeID123 -m "potentially bad"
   ```
