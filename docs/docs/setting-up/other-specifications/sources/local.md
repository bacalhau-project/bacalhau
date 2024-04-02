---
sidebar_label: Local
---
# Local Source Specification

The Local input source allows Bacalhau jobs to access files and directories that are already present on the compute node. This is especially useful for utilizing locally stored datasets, configuration files, logs, or other necessary resources without the need to fetch them from a remote source, ensuring faster job initialization and execution.

## Source Specification Parameters

Here are the parameters that you can define for a Local input source:

- **SourcePath** `(string: <required>)`: The absolute path on the compute node where the Local or file is located. Bacalhau will access this path to read data, and if permitted, write data as well.

- **ReadWrite** `(bool: false)`: A boolean flag that, when set to true, gives Bacalhau both read and write access to the specified Local or file. If set to false, Bacalhau will have read-only access.

### Allow-listing Local Paths

For security reasons, direct access to local paths must be explicitly allowed when running the Bacalhau compute node. This is achieved using the `--allow-listed-local-paths` flag followed by a comma-separated list of the paths, or path patterns, that should be accessible. Each path can be suffixed with permissions as well:

- `:rw` - Read-Write access.
- `:ro` - Read-Only access (default if no suffix is provided).

For instance:

```bash
bacalhau serve --allow-listed-local-paths "/etc/config:rw,/etc/*.conf:ro"
```

### Example

Below is an example of how to define a Local input source in YAML format.

```yaml
InputSources:
  - Source:
      Type: "localDirectory"
      Params:
        SourcePath: "/etc/config"
        ReadWrite: true
    Target: "/config"
```

In this example, Bacalhau is configured to access the Local "/etc/config" on the compute node. The contents of this directory are made available at the "/config" path within the task's environment, with read and write access. Adjusting the `ReadWrite` flag to false would enable read-only access, preventing modifications to the local data from within the Bacalhau task.


### Example (Imperative/CLI)

When using the Bacalhau CLI to define the local input source, you can employ the following imperative approach. Below are example commands demonstrating how to define the local input source with various configurations:

1. **Mount readonly file to `/config`**:
   ```bash
   bacalhau docker run -i file:///etc/config:/config ubuntu ...
   ```

2. **Mount writable file to default `/input`**:
   ```bash
   bacalhau docker run -i file:///var/checkpoints:/myCheckpoints,opt=rw=true ubuntu ...
   ```
