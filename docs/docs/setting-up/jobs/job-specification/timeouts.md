---

sidebar_label: Timeouts
---

# Timeouts Specification

The `Timeouts` object provides a mechanism to impose timing constraints on specific task operations, particularly execution. By setting these timeouts, users can ensure tasks don't run indefinitely and align them with intended durations.

## `Timeouts` Parameters:

- **ExecutionTimeout** `(int: <optional>)`: Defines the maximum duration (in seconds) that a task is permitted to run. A value of zero indicates that there's no set timeout. This could be particularly useful for tasks that function as daemons and are designed to run indefinitely.

Utilizing the `Timeouts` judiciously helps in managing resource utilization and ensures tasks adhere to expected timelines, thereby enhancing the efficiency and predictability of job executions.
