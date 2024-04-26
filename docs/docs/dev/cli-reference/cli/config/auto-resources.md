---
sidebar_label: auto-resources
---

# Command: `config auto-resources`

## Description:

The `bacalhau config auto-resources` command automatically configures compute resource values in the bacalhau node's configuration file based on the hardware resources of the user's machine. This command streamlines the process of resource allocation for jobs, dynamically adjusting settings to align with the capabilities of the machine. It is designed to simplify the task of resource management, ensuring that the node operates efficiently and effectively within the hardware's limits.

Note: The `bacalhau config auto-resources` command intelligently adjusts resource allocation settings based on the specific hardware configuration of your machine, promoting optimal utilization for bacalhau jobs. Due to the dynamic nature of this command, the specific values set in the configuration will vary depending on the available hardware resources of the machine in use. This functionality is particularly beneficial for users who seek to optimize their node's performance without the need for manual calculations of resource limits. It is important for users to understand that these settings will directly impact the number and types of jobs their node can manage at any given time, based on the machine's resource capacity.

## Usage

```bash
bacalhau config auto-resources [flags]
```

## Flags

- `--default-job-percentage int`:
  - Description: Sets the default percentage of resources allocated for each job when specific limits are not defined. Acceptable values range from 1 to 100 (values over 100 are rejected).
  - Default: 75
- `--job-percentage int`:
  - Description: Determines the percentage of resources that can be utilized at one time for a single job. Accept values from 1 to 100 (values over 100 are rejected).
  - Default: 75
- `--queue-job-percentage int`:
  - Description: Specifies the total percentage of resources that the system can allocate for all jobs queued at one time. Accept values from 1 to 100 (values over 100 are accepted).
  - Default: 150
- `--total-percentage int`:
  - Description: Indicates the total percentage of resources that the system can utilize at one time across all jobs. Accept values from 1 to 100 (values over 100 are rejected).
  - Default: 75

## Examples

(Ran on an Apple M1 Max with 10 Cores and 64GB RAM)

1. **Basic Usage**:


   **Command**:

   ```bash
   $ bacalhau config auto-resources
   ```

   **Config File**:

   ```yaml
   node:
       compute:
           capacity:
               defaultjobresourcelimits:
                   cpu: 7500m
                   disk: 568 GB
                   gpu: "0"
                   memory: 52 GB
               jobresourcelimits:
                   cpu: 7500m
                   disk: 568 GB
                   gpu: "0"
                   memory: 52 GB
               queueresourcelimits:
                   cpu: 15000m
                   disk: 1.1 TB
                   gpu: "0"
                   memory: 103 GB
               totalresourcelimits:
                   cpu: 7500m
                   disk: 568 GB
                   gpu: "0"
                   memory: 52 GB
   ```

2. **Queue 500% system resources**:

   **Command**:

   ```bash
   $ bacalhau config auto-resources --queue-job-percentage=500
   ```

   **Config File**:

   ```yaml
   node:
       compute:
           capacity:
               defaultjobresourcelimits:
                   cpu: 7500m
                   disk: 568 GB
                   gpu: "0"
                   memory: 52 GB
               jobresourcelimits:
                   cpu: 7500m
                   disk: 568 GB
                   gpu: "0"
                   memory: 52 GB
               queueresourcelimits:
                   cpu: 50000m
                   disk: 3.8 TB
                   gpu: "0"
                   memory: 344 GB
               totalresourcelimits:
                   cpu: 7500m
                   disk: 568 GB
                   gpu: "0"
                   memory: 52 GB
   ```

3. **With 25% of system resources**:

   **Command**:

   ```bash
   $ bacalhau config auto-resources  --total-percentage=25 --job-percentage=25 --default-job-percentage=25
   ```

   **Config File**:

   ```yaml
   node:
       compute:
           capacity:
               defaultjobresourcelimits:
                   cpu: 2500m
                   disk: 190 GB
                   gpu: "0"
                   memory: 17 GB
               jobresourcelimits:
                   cpu: 2500m
                   disk: 190 GB
                   gpu: "0"
                   memory: 17 GB
               queueresourcelimits:
                   cpu: 15000m
                   disk: 1.1 TB
                   gpu: "0"
                   memory: 103 GB
               totalresourcelimits:
                   cpu: 2500m
                   disk: 190 GB
                   gpu: "0"
                   memory: 17 GB
   ```
