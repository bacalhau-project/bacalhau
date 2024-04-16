---
sidebar_label: list
---

# Command: `node list`

The `bacalhau node list` command is designed to provide users with a comprehensive list of network nodes along with details based on specified flags.

## Description:

The `list` sub-command under the `bacalhau node` category enumerates information about nodes in the network. It supports various filtering, ordering, and output formatting options, allowing users to tailor the output to their needs.

## Usage:

```bash
bacalhau node list [flags]
```

## Flags:

- `-h`, `--help`:

  - Show the help message for the `list` command.

- `--hide-header`:

  - Do not display the column headers in the output.

- `--filter-approval`:

  - Only show nodes with the specified approval status. Valid values are: `approved`, `pending`, `rejected`.

- `--filter-status`:

  - Only show nodes with the specified state. Valid values are: `healthy`, `unhealthy`, `unknown`.

- `--labels string`:

  - Filter nodes based on labels. This follows the filtering format provided by Kubernetes, as shown in their documentation about labels.

- `--limit uint32`:

  - Restrict the number of results displayed.

- `--next-token string`:

  - Provide the next token for pagination.

- `--no-style`:

  - Output the table without any style.

- `--order-by string`:

  - Sort the results based on a specific field. Valid sorting fields are: `id`, `type`, `available_cpu`, `available_memory`, `available_disk`, `available_gpu`.

- `--order-reversed`:

  - Display the results in reverse order.

- `--output format`:

  - Choose the output format. Available options: `table`, `csv`, `json`, `yaml`.
  - Default: `table`.

- `--pretty`:

  - Enhance the visual appeal of the output. This is applicable only to `json` and `yaml` formats.

- `--show strings`:

  - Determine the column groups to be displayed. Acceptable values are: `labels`, `version`, `features`, `capacity`.
  - Default: `labels`, `capacity`.

- `--wide`:
  - Display full values in the output table, without truncation.

## Global Flags:

- `--api-host string`:

  - Specify the host for client-server communication via REST. This gets ignored if the `BACALHAU_API_HOST` environment variable is defined.
  - Default: `"bootstrap.production.bacalhau.org"`.

- `--api-port int`:

  - Specify the port for RESTful communication between client and server. Gets overlooked if the `BACALHAU_API_PORT` environment variable is set.
  - Default: `1234`.

- `--log-mode logging-mode`:

  - Choose the desired log format.
  - Options: `'default', 'station', 'json', 'combined', 'event'`.
  - Default: `'default'`.

- `--repo string`:
  - Point to the directory path of the bacalhau repository.
  - Default: `"`$HOME/.bacalhau"`.

## Examples

1. **Retrieve the list of nodes**:

   Execute the command to get a list of all nodes:

   ```bash
   bacalhau node list
   ```

   Expected output:

   ```plaintext
    ID        TYPE     LABELS                                              CPU     MEMORY      DISK         GPU
    QmTSJgdN  Compute  Architecture=amd64 Operating-System=linux           3.2 /   11.7 GB /   77.8 GB /    1 /
                       git-lfs=True owner=bacalhau                         3.2     11.7 GB     77.8 GB      1
    QmVXwmdZ  Compute  Architecture=amd64 Operating-System=linux           3.2 /   12.5 GB /   77.8 GB /    0 /
                       git-lfs=True owner=bacalhau                         3.2     12.5 GB     77.8 GB      0
    QmXRdLru  Compute  Architecture=amd64 Operating-System=linux           3.2 /   12.5 GB /   78.0 GB /    0 /
                       git-lfs=True owner=bacalhau                         3.2     12.5 GB     78.0 GB      0
    ... [Additional nodes information] ...
   ```

1. **Filter the list of nodes by labels**:

   Execute the command to get a list of nodes with specific labels:

   ```bash
   bacalhau node list --labels "Operating-System=linux,owner=bacalhau"
   ```

   Expected output:

   ```plaintext
   ID        TYPE     LABELS                                              CPU     MEMORY      DISK         GPU
   QmTSJgdN  Compute  Architecture=amd64 Operating-System=linux           3.2 /   11.7 GB /   77.8 GB /    1 /
                      git-lfs=True owner=bacalhau                         3.2     11.7 GB     77.8 GB      1
   ... [Additional nodes information] ...
   ```

1. **Order the list of nodes by available memory**:

   Execute the command to get the list of nodes ordered by available memory:

   ```bash
   bacalhau node list --order-by available_memory
   ```

   Expected output:

   ```plaintext
   ID        TYPE     LABELS                                              CPU     MEMORY      DISK         GPU
   QmVXwmdZ  Compute  Architecture=amd64 Operating-System=linux           3.2 /   12.5 GB /   77.8 GB /    0 /
                      git-lfs=True owner=bacalhau                         3.2     12.5 GB     77.8 GB      0
   ... [Additional nodes information] ...
   ```

1. **Limit the number of nodes displayed and output in JSON format**:

   Execute the command to get a limited list of nodes in JSON format:

   ```bash
   bacalhau node list  --limit 3 --output json --pretty
   ```

   Expected output:

   ```json
   [
     {
       "PeerInfo": {
         "ID": "QmTSJgdN7zCPAqBCkmdsdpFbiJV8bJ6zhoxK9N5xfar1sz",
         ... [Additional node details] ...
       },
       ... [Other nodes] ...
     }
   ]
   ```
