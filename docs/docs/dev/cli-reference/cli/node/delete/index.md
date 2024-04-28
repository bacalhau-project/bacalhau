---
sidebar_label: delete
---

# Command: `node delete`

The `bacalhau node delete` command offers administrators the ability to remove a node from the cluster using its name.

## Description:

Using the `delete` sub-command, administrators can remove a node from the list of available compute nodes in the cluster. This feature is necessary for the management of the infrastructure.

## Usage:

```bash
bacalhau node delete [id] [flags]
```

## Flags:

- `[id]`:

  - The unique identifier of the node you wish to describe.

- `-h`, `--help`:

  - Displays the help documentation for the `describe` command.

- `-m message`:

  - A message to be attached to the deletion action.

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

1. Delete the Node with ID `nodeID123`:

   ```bash
   bacalhau node delete nodeID123
   ```

2. Delete a Node with an audit message:

   ```bash
   bacalhau node delete nodeID123 -m "bad actor"
   ```
