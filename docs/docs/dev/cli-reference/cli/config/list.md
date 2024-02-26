---
sidebar_label: list
---

# Command: `config list`

## Description

The `bacalhau config list` command lists the configuration keys and values of the bacalhau node. This command is useful for understanding how configuration keys map to their respective values, aiding in the use of the `bacalhau config set` command.

Note: Configuration values displayed by this command represent the settings that will be applied when the bacalhau node is next restarted. It is important to note that these values may not reflect the current operational configuration of an active bacalhau node. The displayed configuration is relevant and accurate for a node that is either not currently running or that has been restarted after the execution of this command.

## Usage

```bash
bacalhau config list [flags]
```

## Flags

- `-h`, `--help`:
    - Description: Displays help information for the `list` sub-command.
- `--hide-header`:
    - Description: Do not print the column headers when displaying the results.
    - Default: `false`
-  `--no-style`:
    - Description: Removes all styling from the table output, displaying raw data.
    - Default: `false`
- `--output format`:
    - Description: Determines the format in which the output is displayed. Available formats include Table, JSON, and YAML.
    - Options: `json`, `yaml`, `table`
    - Default: `table`
- `--pretty`:
    - Description: Formats the output for enhanced readability. This flag is relevant only when using JSON or YAML output formats.
    - Default: `true`
- `--wide`:
    - Description: Prints full values in the table results without truncating any information.
    - Default: `false`

## Examples

### Listing the Bacalhau nodes configuration settings

1. **Basic Usage**:

   **Command**:

   ```bash
   $ bacalhau config list
   ```

   **Output**:

   ```bash
    KEY          VALUE
    <key_name> <key_value>
   ...
   ```

2. **Output in JSON format**:

   **Command**:

   ```bash
   $ bacalhau config list --output json --pretty
   ```

   **Output**:

   ```json
   [
     {
       "Key": "<key_name>",
       "Value": <key_value>
     },
     ...
   ]
   ```
