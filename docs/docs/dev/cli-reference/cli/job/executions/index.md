---
sidebar_label: executions
---
# Command: `job executions`

## Description

The `bacalhau job executions` command retrieves a list of executions for a specific job based on its ID. This can be essential when tracking the various runs and their respective states for a particular job.

## Usage

```
bacalhau job executions [id] [flags]
```


## Flags

- `-h`, `--help`:
    - Description: Display help for the `executions` command.

- `--hide-header`:
    - Description: Do not print the column headers when displaying the results.

- `--limit uint32`:
    - Description: Restricts the number of results returned.
    - Default: `20`

- `--next-token string`:
    - Description: Uses the specified token for pagination. Useful for fetching the next set of results.

- `--no-style`:
    - Description: Removes all styling from the table output, displaying raw data.

- `--order-by string`:
    - Description: Orders results based on a specific field. Valid fields are: `modify_time`, `create_time`, `id`, and `state`.

- `--order-reversed`:
    - Description: Reverses the order of the results. Useful in conjunction with `--order-by`.

- `--output format`:
    - Description: Specifies the desired output format for the command. Supported values are `table`, `csv`, `json`, and `yaml`.
    - Default: `table`

- `--pretty`:
    - Description: Pretty prints the output. This option is applicable only to `json` and `yaml` output formats.

- `--wide`:
    - Description: Prints full values in the table results without truncating any information.

## Global Flags

- `--api-host string`:
    - Description: Specifies the host for the client and server to communicate through via REST. If the `BACALHAU_API_HOST` environment variable is set, this flag will be ignored.
    - Default: `bootstrap.production.bacalhau.org`

- `--api-port int`:
    - Description: Determines the port for the client and server to communicate on using REST. If the `BACALHAU_API_PORT` environment variable is set, this flag will be ignored.
    - Default: `1234`

- `--log-mode logging-mode`:
    - Description: Specifies the desired log format. Supported values include `default`, `station`, `json`, `combined`, and `event`.
    - Default: `default`

- `--repo string`:
    - Description: Defines the path to the bacalhau repository.
    - Default: `$HOME/.bacalhau`


## Examples

1. **List executions for a specific Job**:

   ```bash
   bacalhau job executions j-6f2bf0ea-ebcd-4490-899a-9de9d8d95881
   ```

   Expected output:

   ```bash
   CREATED   MODIFIED  ID          NODE ID   REV.  COMPUTE    DESIRED  COMMENT
                                                   STATE      STATE
   16:46:03  16:46:04  e-99362435  QmTSJgdN  6     Completed  Stopped
   16:46:03  16:46:04  e-75dd20bb  QmXRdLru  6     Completed  Stopped
   16:46:03  16:46:04  e-03870df5  QmVXwmdZ  6     Completed  Stopped
   ```

2. **Order executions by state for a specific job**:

   Execute the command:

   ```bash
   bacalhau job executions j-6f2bf0ea-ebcd-4490-899a-9de9d8d95881 --order-by state
   ```

   Expected output:

   ```bash
   CREATED   MODIFIED  ID          NODE ID   REV.  COMPUTE    DESIRED  COMMENT
                                                   STATE      STATE
   16:46:03  16:46:04  e-03870df5  QmVXwmdZ  6     Completed  Stopped
   16:46:03  16:46:04  e-75dd20bb  QmXRdLru  6     Completed  Stopped
   16:46:03  16:46:04  e-99362435  QmTSJgdN  6     Completed  Stopped
   ```

3. **List executions with YAML output**:
    ```bash
    bacalhau job executions j-6f2bf0ea-ebcd-4490-899a-9de9d8d95881 --output yaml
    ```

   Expected output:

   ```yaml
   ... [The YAML formatted output] ...
   ```
