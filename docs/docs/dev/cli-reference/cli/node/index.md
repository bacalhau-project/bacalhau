# command: `node`

## Description

The `bacalhau node` command provides a set of sub-commands to query and manage node-related information within Bacalhau. With these tools, users can access specific details about nodes, list all network nodes, and more.

## Usage

```
bacalhau node [command]
```

## Available Commands

1. **[approve](./approve)**:

   - Description: Approves a single node to join the cluster.
   - Usage:

     ```bash
     bacalhau node approve
     ```

1. **[delete](./delete)**:

   - Description: Deletes a node from the cluster using its ID.
   - Usage:
     ```bash
     bacalhau node delete
     ```

1. **[describe](./describe)**:

   - Description: Retrieves detailed information of a node using its ID.
   - Usage:
     ```bash
     bacalhau node describe
     ```

1. **[list](./list)**:

   - Description: Lists the details of all nodes present in the network.
   - Usage:
     ```bash
     bacalhau node list
     ```

1. **[reject](./reject)**:

- Description: Reject a specific node's request to join the cluster.
- Usage:
  ```bash
  bacalhau node reject
  ```

For comprehensive details on any of the sub-commands, run:

```bash
bacalhau node [command] --help
```

## Flags

- `-h`, `--help`:
  - Description: Shows the help information for the `node` command.

## Global Flags

- `--api-host string`:

  - Description: Specifies the host for RESTful communication between the client and server. The flag will be ignored if the `BACALHAU_API_HOST` environment variable is set.
  - Default: `bootstrap.production.bacalhau.org`

- `--api-port int`:

  - Description: Designates the port for RESTful communication. The flag will be bypassed if the `BACALHAU_API_PORT` environment variable is active.
  - Default: `1234`

- `--log-mode logging-mode`:

  - Description: Chooses the preferred log format. Available choices are: `default`, `station`, `json`, `combined`, and `event`.
  - Default: `default`

- `--repo string`:
  - Description: Specifies the path to the bacalhau repository.
  - Default: `/Users/walid/.bacalhau`
