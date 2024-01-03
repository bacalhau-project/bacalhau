---
sidebar_label: history
---
# Command: `job history`

## Description

The `bacalhau job history` command lists the history events of a specific job based on its ID. This feature allows users to track changes, executions, and other significant milestones associated with a particular job.

## Usage

```
bacalhau job history [id] [flags]
```


## Flags

- `--event-type string`:
    - Description: Specifies the type of history events to retrieve. Available options include `all`, `job`, and `execution`.
    - Default: `all`

- `--execution-id string`:
    - Description: Filters results by a specific execution ID.

- `-h`, `--help`:
    - Description: Display help for the `history` command.

- `--hide-header`:
    - Description: Opts out of printing the column headers in the results.

- `--limit uint32`:
    - Description: Limits the number of results returned.

- `--next-token string`:
    - Description: Uses the provided token for pagination.

- `--no-style`:
    - Description: Strips all styling from the table output.

- `--node-id string`:
    - Description: Filters the results by a specific node ID.

- `--order-by string`:
    - Description: Organizes results based on a chosen field.

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

1. **Retrieve the history of a specific job**:

   Execute the command to get the job history:

   ```bash
   bacalhau job history j-6f2bf0ea-ebcd-4490-899a-9de9d8d95881
   ```

   Expected output:

   ```plaintext
   TIME      LEVEL           EXEC. ID    ...     NEW STATE          COMMENT
   ... [The output rows like the ones you've shown] ...
   16:46:04  JobLevel                              2     Pending            Completed
   ```

1. **Filter the history by event type**:

   Filter the job history by the event type:

   ```bash
   bacalhau job history j-6f2bf0ea-ebcd-4490-899a-9de9d8d95881 --event-type job
   ```

   Expected output:

   ```plaintext
   TIME      LEVEL     EXEC. ID  NODE ID  REV.  PREVIOUS STATE  NEW STATE  COMMENT
   16:46:03  JobLevel                     1     Pending         Pending    Job created
   16:46:04  JobLevel                     2     Pending         Completed
   ```

1. **Filter the history by execution ID**:

   Filter the job history by a specific execution ID:

   ```bash
   bacalhau job history j-6f2bf0ea-ebcd-4490-899a-9de9d8d95881 --execution-id e-99362435
   ```

   Expected output:

   ```plaintext
   TIME      LEVEL           EXEC. ID    ...     NEW STATE          COMMENT
   ... [The output rows for the specific execution ID] ...
   16:46:04  ExecutionLevel  e-99362435  QmTSJgdN  6     BidAccepted        Completed
   ```

1. **Retrieve the history in YAML format**:

   Get the job history in YAML format:

   ```bash
   bacalhau job history j-6f2bf0ea-ebcd-4490-899a-9de9d8d95881 --output yaml
   ```

   Expected output:

   ```yaml
   ... [The YAML formatted output] ...
   ```
