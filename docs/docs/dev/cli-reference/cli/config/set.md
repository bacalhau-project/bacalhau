---
sidebar_label: set
---

# Command: `config set`

## Description:

The `bacalhau config set` command sets a value in the bacalhau node's configuration file. This command is used to modify the configuration file that the bacalhau node will reference for its settings. Key names in the configuration are case-insensitive. Additionally, the command validates the value being set based on the type of the configuration key, ensuring that only appropriate and valid configurations are applied.

 Note: Changes made using this command will be applied to the configuration file, but they do not immediately affect the running configuration of an active bacalhau node. The modifications will take effect only after the node is restarted.

## Usage

```bash
bacalhau config set <key> <value>
```

## Flags

- `-h`, `--help`:
    - Description: Displays help information for the `set` sub-command.

## Examples

### Configuring the Server API Port Value

```bash
$ bacalhau config set node.serverapi.port 9999
$ bacalhau config list | grep serverapi.port
 node.serverapi.port                                             9999
$ cat ~/.bacalhau/config.yaml
node:
    serverapi:
        port: 9999
```

### Configuring the Logging Mode Value

```bash
$ bacalhau config set node.loggingmode json
$ bacalhau config list | grep loggingmode
 node.loggingmode                                                json
$ cat ~/.bacalhau/config.yaml
node:
    loggingmode: json
```

### Multiple Set commands append to the file

```bash
$ bacalhau config set node.serverapi.port 9999
$ bacalhau config set node.serverapi.host 0.0.0.0
$ bacalhau config set node.loggingmode json
$ cat ~/.bacalhau/config.yaml
node:
    loggingmode: json
    serverapi:
        host: 0.0.0.0
        port: 9999
```

### Set command value validation

**Example of invalid logging mode value**
```bash
$ bacalhau config set node.loggingmode some-invalid-value
Error: setting "node.loggingmode": "some-invalid-value" is an invalid log-mode (valid modes: ["default" "station" "json" "combined" "event"])
```

**Example of invalid time duration value**
```bash
$ bacalhau config set node.volumesizerequesttimeout 10days
Error: setting "node.volumesizerequesttimeout": time: unknown unit "days" in duration "10days"

```
