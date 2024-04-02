---
sidebar_label: IPFS
---

# IPFS Source Specification

The IPFS Input Source enables users to easily integrate data hosted on the [InterPlanetary File System (IPFS)](https://ipfs.tech) into Bacalhau jobs. By specifying the Content Identifier (CID) of the desired IPFS file or directory, users can have the content fetched and made available in the task's execution environment, ensuring efficient and decentralized data access.

## Source Specification Parameters

Here are the parameters that you can define for an IPFS input source:

- **CID** `(string: <required>)`: The Content Identifier that uniquely pinpoints the file or directory on the IPFS network. Bacalhau retrieves the content associated with this CID for use in the task.

### Example

Below is an example of how to define an IPFS input source in YAML format.

```yaml
InputSources:
  - Source:
      Type: "ipfs"
      Params:
        CID: "QmY7Yh4UquoXHLPFo2XbhXkhBvFoPwmQUSa92pxnxjY3fZ"
  - Target: "/data"
```

In this configuration, the data associated with the specified CID is fetched from the IPFS network and made available in the task's environment at the "/data" path.

### Example (Imperative/CLI)

Utilizing IPFS as an input source in Bacalhau via the CLI is straightforward. Below are example commands that demonstrate how to define the IPFS input source:

1. **Mount an IPFS CID to the default `/inputs` directory**:
   ```bash
   bacalhau docker run -i ipfs://QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72 ubuntu ...
   ```

2. **Mount an IPFS CID to a custom `/data` directory**:
   ```bash
   bacalhau docker run -i ipfs://QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/data ubuntu ...
   ```

These commands provide a seamless mechanism to fetch and mount data from IPFS directly into your task's execution environment using the Bacalhau CLI.
