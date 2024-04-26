---
sidebar_label: describe
---
# Command: `job describe`

## Description

The `bacalhau job describe` command provides a detailed description of a specific job in YAML format. This description can be particularly useful when wanting to understand the attributes and current status of a specific job. To list all available jobs, the `bacalhau job list` command can be used.

## Usage

```
bacalhau job describe [id] [flags]
```

## Flags

- `-h`, `--help`:
    - Description: Display help for the `describe` command.

- `--output format`:
    - Description: Specifies the desired output format for the command. Supported values are `json` and `yaml`.
    - Default: `yaml`

- `--pretty`:
    - Description: Pretty prints the output. This option is applicable only to `json` and `yaml` output formats.

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

1. **Describe a Job with Full ID**:
    ```bash
    bacalhau job describe j-e3f8c209-d683-4a41-b840-f09b88d087b9
    ```

2. **Describe a Job with Shortened ID**:
    ```bash
    bacalhau job describe j-47805f5c
    ```

3. **Describe a Job with JSON Output**:
    ```bash
    bacalhau job describe --output json --pretty j-b6ad164a
    ```
