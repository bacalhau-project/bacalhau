---
sidebar_label: Resources
---

# Resources Specification

The `Resources` provides a structured way to detail the computational resources a `Task` requires. By specifying these requirements, you ensure that the task is scheduled on a node with adequate resources, optimizing performance and avoiding potential issues linked to resource constraints.

## `Resources` Parameters:

- **CPU** `(string: <optional>)`: Defines the CPU resources required for the task. Units can be specified in cores (e.g., `2` for 2 CPU cores) or in milliCPU units (e.g., `250m` or `0.25` for 250 milliCPU units). For instance, if you have half a CPU core, you can represent it as `500m` or `0.5`.

- **Memory** `(string: <optional>)`: Highlights the amount of RAM needed for the task. You can specify the memory in various units such as:
    - `Kb` for Kilobytes
    - `Mb` for Megabytes
    - `Gb` for Gigabytes
    - `Tb` for Terabytes

- **Disk** `(string: <optional>)`: States the disk storage space needed for the task. Similarly, the disk space can be expressed in units like `Gb` for Gigabytes, `Mb` for Megabytes, and so on. As an example, `10Gb` indicates 10 Gigabytes of storage space.

- **GPU** `(string: <optional>)`: Denotes the number of GPU units required. For example, `2` signifies the requirement of 2 GPU units. This is crucial for tasks involving heavy computational processes, machine learning models, or tasks that leverage GPU acceleration.
