---
sidebar_label: logs
---
# Command: `job logs`

## Description

The `bacalhau job logs` command allows users to retrieve logs from a job that has been previously submitted. This command is useful for tracking and debugging the progress and state of a running or completed job.

## Usage

```
bacalhau job logs [id] [flags]
```

## Flags

- `-f`, `--follow`:
    - Description: This flag allows the user to follow the logs in real-time after fetching the current logs. It provides a continuous stream of log updates, similar to `tail -f` in Unix-like systems.

- `-h`, `--help`:
    - Description: Display help information for the `logs` command.

## Global Flags

- `--api-host string`:
    - Description: Specifies the host for the client and server to communicate through REST. This flag is disregarded if the `BACALHAU_API_HOST` environment variable is set.
    - Default: `bootstrap.production.bacalhau.org`

- `--api-port int`:
    - Description: Sets the port for RESTful communication between the client and server. If the `BACALHAU_API_PORT` environment variable is available, this flag is ignored.
    - Default: `1234`

- `--log-mode logging-mode`:
    - Description: Determines the desired log format. Available options include `default`, `station`, `json`, `combined`, and `event`.
    - Default: `default`

- `--repo string`:
    - Description: Specifies the path to the bacalhau repository.
    - Default: `$HOME/.bacalhau`


## Examples

1. **Display Logs for a Previously Submitted Job with Full ID**:

   **Command:**

   ```bash
   bacalhau job logs j-51225160-807e-48b8-88c9-28311c7899e1
   ```

   **Expected Output:**

   ```plaintext
   [2023-09-24 09:01:32] INFO - Application started successfully.
   [2023-09-24 09:01:33] DEBUG - Initializing database connections.
   [2023-09-24 09:01:35] WARN - API rate limit approaching.
   [2023-09-24 09:02:01] ERROR - Failed to retrieve data from endpoint: /api/v1/data.
   [2023-09-24 09:05:00] INFO - Data sync completed with 4500 new records.
   ```

2. **Follow Logs in Real-Time**:

   **Command:**

   ```bash
   bacalhau job logs --follow j-51225160-807e-48b8-88c9-28311c7899e1
   ```

   **Expected Output**:

   ```plaintext
   [2023-09-24 11:30:02] INFO - User 'john_doe' logged in successfully.
   [2023-09-24 11:30:15] DEBUG - Fetching data from cache for key: userSettings_john_doe.
   [2023-09-24 11:31:05] WARN - High memory usage detected: 85% of allocated resources.
   ... [Logs continue to appear in real-time] ...
   ```

3. **Display Logs Using a Shortened ID**:

   **Command:**

   ```bash
   bacalhau job logs j-ebd9bf2f
   ```

   **Expected Output:**

   ```plaintext
   [2023-09-24 10:15:12] INFO - Application initialization sequence started.
   [2023-09-24 10:15:13] DEBUG - Loading configurations from /config/app.json.
   [2023-09-24 10:15:14] INFO - Connected to message broker successfully.
   [2023-09-24 10:16:00] ERROR - Failed to send email notification to user@example.com.
   ```
