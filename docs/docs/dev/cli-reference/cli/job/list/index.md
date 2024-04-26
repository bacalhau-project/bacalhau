---
sidebar_label: list
---
# Command: `job list`

## Description

The `bacalhau job list` command provides a listing of all submitted jobs. This command offers an overview of all tasks and processes registered in the system, allowing users to monitor and manage their jobs effectively.

## Usage

```
bacalhau job list [flags]
```


## Flags

- `-h`, `--help`:
    - Description: Display help for the `list` command.

- `--hide-header`:
    - Description: Opts out of printing the column headers in the results.

- `--labels string`:
    - Description: Filters jobs by labels. It's designed to function similar to Kubernetes label selectors.
    - Default: `bacalhau_canary != true`

- `--limit uint32`:
    - Description: Limits the number of results returned.
    - Default: `10`

- `--next-token string`:
    - Description: Uses the provided token for pagination.

- `--no-style`:
    - Description: Strips all styling from the table output.

- `--order-by string`:
    - Description: Organizes results based on a chosen field. Valid fields are `id` and `created_at`.

- `--order-reversed`:
    - Description: Reverses the order of the displayed results.

- `--output format`:
    - Description: Dictates the desired output format for the command. Options are `table`, `csv`, `json`, and `yaml`.
    - Default: `table`

- `--pretty`:
    - Description: Offers a more visually pleasing output for `json` and `yaml` formats.

- `--wide`:
    - Description: Presents full values in the table results, preventing truncation.

## Global Flags

- `--api-host string`:
    - Description: Defines the host for client-server communication via REST. Overridden by the `BACALHAU_API_HOST` environment variable, if set.
    - Default: `bootstrap.production.bacalhau.org`

- `--api-port int`:
    - Description: Sets the port for RESTful communication between the client and server. The `BACALHAU_API_PORT` environment variable takes precedence if set.
    - Default: `1234`

- `--log-mode logging-mode`:
    - Description: Designates the desired log format. Options include `default`, `station`, `json`, `combined`, and `event`.
    - Default: `default`

- `--repo string`:
    - Description: Points to the bacalhau repository location.
    - Default: `$HOME/.bacalhau`


## Examples

1. **List all jobs**:

   Execute the command to list all the jobs:

   ```bash
   bacalhau job list
   ```

   Expected output:

   ```plaintext
   CREATED   ID          JOB     TYPE   STATE
   08:19:07  d78a4cb4    docker  batch  Completed
   04:17:21  e45f31a7    docker  batch  Completed
   04:53:50  f4993f62    docker  batch  Completed
   ... (trimmed for brevity) ...
   ```

2. **Limit the list to the last two jobs**:

   Limit the list to display only the last two jobs:

   ```bash
   bacalhau job list --limit 2
   ```

   Expected output:

   ```plaintext
   CREATED   ID          JOB     TYPE   STATE
   03:14:16  19a26187    docker  batch  Completed
   21:47:21  2a53a13b    docker  batch  Completed
   ```

3. **Order the list by creation date in descending order**:

   Order the jobs by their creation date in a descending manner:

   ```bash
   bacalhau job list --order-by created_at --order-reversed
   ```

   Expected output:

   ```plaintext
   CREATED   ID          JOB     TYPE   STATE
   17:44:16  90e14efd    docker  batch  Completed
   17:44:08  8204570c    docker  batch  Completed
   17:43:50  f196521d    docker  batch  Completed
   ... (trimmed for brevity) ...
   ```

4. **Filter the jobs by specific labels**:

   Display jobs that have specific labels:

   ```bash
   bacalhau job list --labels "region in (us-east-1, us-east-2),env = prod"
   ```

   Expected output:

   ```plaintext
   ... (filtered jobs) ...
   ```

5. **Display the list in JSON format with pretty printing**:

   Get a limited list of jobs in a formatted JSON output:

   ```bash
   bacalhau job list --limit 3 --output json --pretty
   ```

   Expected output:

   ```plaintext
   ... [The JSON formatted output] ...
   ```
