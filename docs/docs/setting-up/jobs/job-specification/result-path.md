---
sidebar_label: ResultPath
---

# ResultPath Specification

A `ResultPath` denotes a specific location within a `Task` that contains meaningful output or results. By specifying a `ResultPath`, you can pinpoint which files or directories are essential and should be retained or published after the task's execution.

## `ResultPath` Parameters:

- **Name**: A descriptive label or identifier for the result, allowing for easier referencing and understanding of the output's nature or significance.

- **Path**: Specifies the exact location, either a file or a directory, within the task's environment where the result or output is stored. This ensures that after the task completes, the critical data at this path can be accessed, retained, or published as necessary.
