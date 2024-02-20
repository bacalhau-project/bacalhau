---
sidebar_label: Local
---

# Local Publisher Specification

Bacalhau's Local Publisher provides a useful option for storing task results on the compute node, allowing for ease of access and retrieval for testing or trying our Bacalhau.

:::danger

The Local Publisher should not be used for Production use as it is not a reliable storage option. For production use, we recommend using a more reliable option such as an S3-compatible storage service.
:::

## Local Publisher Parameters
The local publisher requires no specific parameters to be defined in the publisher specification. The user only needs to indicate the publisher type as "local", and Bacalhau handles the rest. Here is an example of how to set up a Local Publisher in a job specification.

```yaml
Publisher:
  Type: local
```

## Published Result Specification
Once the job is executed, the results are published to the local compute node, and stored as compressed tar file, which can be accessed and retrieved over HTTP from the command line using the `get` command. TAhis will download and extract the contents for the user from the remove compute node.

### Result Parameters
- URL `(string)`: This is the HTTP URL to the results of the computation, which is hosted on the compute node where it ran.
Here's a sample of how the published result might appear:

```yaml
PublishedResult:
  Type: local
  Params:
    URL: "http://192.168.0.11:6001/e-c4b80d04-ff2b-49d6-9b99-d3a8e669a6bf.tgz"
```

In this example, the task results will be stored on the compute node, and can be referenced and retrieved using the specified URL.


## Caveats

- By default the compute node will attempt to use a public address for the HTTP server delivering task output, but there is no guarantee that the compute node is accessible on that address. If the compute node is behind a NAT or firewall, the user may need to manually specify the address to use for the HTTP server in the `config.yaml` file.
- There is no lifecycle management for the content stored on the compute node. The user is responsible for managing the content and ensuring that it is removed when no longer needed before the compute node runs out of disk space.
- If the address/port of the compute node changes, then previously stored content will no longer be accessible. The user will need to manually update the address in the `config.yaml` file and re-publish the content to make it accessible again.
