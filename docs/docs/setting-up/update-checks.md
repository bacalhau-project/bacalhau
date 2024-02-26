---
sidebar_label: 'Update Checking'
sidebar_position: 6
---

# Automatic Update Checking

Bacalhau has an update checking service to automatically detect whether a newer version of the software is available.

Users who are both running CLI commands and operating nodes will be regularly informed that a new release can be downloaded and installed.


## For clients

Bacalhau will run an update check regularly when client commands are executed. If an update is available, explanatory text will be printed at the end of the command.

To force a manual update check, run the `bacalhau version` command, which will explicitly list the latest software release alongside the server and client versions.

```shell
bacalhau version

Expected Output
 CLIENT  SERVER  LATEST  UPDATE MESSAGE
 v1.2.0  v1.2.0  v1.2.0
```

## For node operators

Bacalhau will run an update check regularly as part of the normal operation of the node.

If an update is available, an INFO level message will be printed to the log.

## Configuring checks

Bacalhau has some configuration options for controlling how often checks are performed. By default, an update check will run no more than once every 24 hours. Users can opt out of automatic update checks using the configuration described below.

| Config property | Environment variable | Default value | Meaning |
|---|---|---|---|
| Update.SkipChecks | `BACALHAU_UPDATE_SKIPCHECKS` | False | If true, no update checks will be performed. As an environment variable, set to `"1"`, `"t"` or `"true"`. |
| Update.CheckFrequency | `BACALHAU_UPDATE_CHECKFREQUENCY` | 24 hours | The minimum amount of time between automated update checks. Set as any duration of hours, minutes or seconds, e.g. `24h` or `10m`. |
| Update.CheckStatePath | `BACALHAU_UPDATE_CHECKSTATEPATH` | $BACALHAU_DIR/update.json | An absolute path where Bacalhau should store the date and time of the last check. |

:::info
It's important to note that disabling the automatic update checks may lead to potential issues, arising from mismatched versions of different actors within Bacalhau.
:::

To output update check config, run `bacalhau config list`:

```shell
bacalhau config list

Expected Output
...
update.checkfrequency                                           24h0m0s
update.checkstatepath                                           /home/user/.bacalhau/update.json
update.skipchecks                                               false
...
```
