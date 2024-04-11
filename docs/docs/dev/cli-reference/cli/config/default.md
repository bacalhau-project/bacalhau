---
sidebar_label: default
---

# Command: `config default`

## Description:

The `bacalhau config default` command prints the default configuration of a bacalhau node to the standard output (stdout). This command is advantageous for users to view the baseline settings a bacalhau node will use in the absence of any user-defined configuration changes. It provides a clear view of the default operational parameters of the node, aiding users in understanding and customizing their configuration from a known baseline.

Note: The output of this command shows the initial default settings for a new bacalhau node and is useful for understanding the foundational settings for customization. To apply these default settings, you can redirect the output to your configuration file using `bacalhau config default > ~/.bacalhau/config.yaml`, which overwrites your current configuration file with the default settings. However, if you wish to always use the latest default settings, especially if the defaults are updated over time, consider deleting your existing configuration file (e.g., `~/.bacalhau/config.yaml`). This approach ensures that your bacalhau node uses the most current defaults, circumventing potential discrepancies between the latest defaults and those captured in an older configuration file created with `bacalhau config default`.

## Usage

```bash
bacalhau config default
```

## Flags

- `-h`, `--help`:
  - Description: Displays help information for the `list` sub-command.
- `--path`:
  - Description: Sets path dependent config fields
  - Default: `$HOME/.bacalhau`

## Examples

### Redirecting Default Configuration to a File

```bash
$ bacalhau config default > ~/.bacalhau/config.yaml
# This command redirects the default configuration output directly into the bacalhau configuration file at ~/.bacalhau/config.yaml, effectively resetting it to default settings.
```
