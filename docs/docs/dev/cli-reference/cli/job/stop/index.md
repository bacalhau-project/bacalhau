---
sidebar_label: stop
---
# Command: `job stop`

## Description

The `bacalhau job stop` command allows users to terminate a previously submitted job. This is useful in scenarios where there's a need to halt a running job, perhaps due to misconfiguration or changed priorities.

## Usage

```
bacalhau job stop [id] [flags]
```


## Flags

- `--quiet`:
    - Description: If provided, the command will not display any output, neither to the standard output (stdout) nor to the standard error (stderr).

- `-h`, `--help`:
    - Description: Displays help information for the `stop` command.

## Global Flags

- `--api-host string`:
    - Description: Specifies the host used for RESTful communication between the client and server. The flag is disregarded if `BACALHAU_API_HOST` environment variable is set.
    - Default: `bootstrap.production.bacalhau.org`

- `--api-port int`:
    - Description: Determines the port for REST communication. If `BACALHAU_API_PORT` environment variable is set, this flag will be ignored.
    - Default: `1234`

- `--log-mode logging-mode`:
    - Description: Selects the desired log format. Options include: `default`, `station`, `json`, `combined`, and `event`.
    - Default: `default`

- `--repo string`:
    - Description: Defines the path to the bacalhau repository.
    - Default: `$HOME/.bacalhau`


## Examples

1. **Stop a Specific Job**:

   If you wish to halt the execution of a job, you can utilize the `stop` command. Here's how you can achieve that:

   **Command:**

   ```bash
   bacalhau job stop j-10eb97de-14cd-4db4-96ec-561bb943309a
   ```

   **Expected Output:**

   ```plaintext
   Checking job status

   	Connecting to network  ................  done ✅  0.0s
   	  Verifying job state  ................  done ✅  0.2s
   	          Stopping job ................  done ✅  0.1s

   Job stop successfully submitted with evaluation ID: 397fd425-8b1a-491e-952a-0632492e7ece
   ```

2. **Silently Stop a Job**:

   If you prefer to terminate a job without seeing any verbose feedback or messages, the `--quiet` option can be used.

   **Command:**

   ```bash
   bacalhau job stop j-63b5ec0c-b5bf-4398-a152-b46c07abe52a --quiet
   ```

   **Expected Output:**

   ```plaintext
   [No output displayed as the operation is run quietly.]
   ```
