---
sidebar_label: version
---
# Command: `agent version`

The `bacalhau agent version` command is used to obtain the version of the bacalhau agent.

## Description:
Using this command, users can quickly retrieve the version of the agent, allowing them to confirm the specific release of the software they are using.

## Usage:
```bash
bacalhau agent version [flags]
```

## Flags:
- **`-h`, `--help`**:
  - Show help for the `version` command.

- **`--output format`**:
  - Defines the output format of the command's results. Accepted formats include "json" and "yaml".

- **`--pretty`**:
  - Used for pretty printing the output, enhancing readability. This flag is applicable only for the "json" and "yaml" output formats.

## Global Flags:

- **`--api-host string`**:
  - Designates the host for client-server communication via REST. If the `BACALHAU_API_HOST` environment variable is present, this flag will be disregarded.
  - Default: `"bootstrap.production.bacalhau.org"`

- **`--api-port int`**:
  - Defines the port for client-server communication through REST. This flag becomes irrelevant if the `BACALHAU_API_PORT` environment variable is specified.
  - Default: `1234`

- **`--log-mode logging-mode`**:
  - Specifies the desired logging format.
  - Options: `'default','station','json','combined','event'`
  - Default: `'default'`

- **`--repo string`**:
  - Indicates the path to the bacalhau repository.
  - Default: `"`$HOME/.bacalhau"`

## Examples

1. **Retrieve the agent version**:

   Execute the command to get the agent version:

   ```bash
   bacalhau agent version
   ```

   Expected output:

   ```bash
   Bacalhau v0.0.0-xxxxxxx
   BuildDate 2023-09-22 16:03:44 +0000 UTC
   GitCommit 0fe81cb488f666845ac72c73a4b804aaa658e511
   ```

2. **Retrieve the agent version in JSON format**:

   ```bash
   bacalhau agent version --output json
   ```

   Expected output:

   ```bash
   {"major":"0","minor":"0","gitversion":"v0.0.0-xxxxxxx","gitcommit":"0fe81cb488f666845ac72c73a4b804aaa658e511","builddate":"2023-09-22T16:03:44Z","goos":"linux","goarch":"amd64"}
   ```

3. **Retrieve the agent version in Pretty-printed JSON format**:

   ```bash
   bacalhau agent version --output json --pretty
   ```

   Expected output:

   ```bash
   {
     "major": "0",
     "minor": "0",
     "gitversion": "v0.0.0-xxxxxxx",
     "gitcommit": "0fe81cb488f666845ac72c73a4b804aaa658e511",
     "builddate": "2023-09-22T16:03:44Z",
     "goos": "linux",
     "goarch": "amd64"
   }
   ```
