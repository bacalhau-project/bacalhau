---
sidebar_label: URL
---

# URL Source Specification

The URL Input Source provides a straightforward method for Bacalhau jobs to access and incorporate data available over HTTP/HTTPS. By specifying a URL, users can ensure the required data, whether a single file or a web page content, is retrieved and prepared in the task's execution environment, enabling direct and efficient data utilization.

## Source Specification Parameters

Here are the parameters that you can define for a URL input source:

- **URL** `(string: <required>)`: The HTTP/HTTPS URL pointing directly to the file or web content you want to retrieve. The content accessible at this URL will be fetched and made available in the task’s environment.

### Example

Below is an example of how to define a URL input source in YAML format.

```yaml
InputSources:
  - Source:
      Type: "urlDownload"
      Params:
        URL: "https://example.com/data/file.txt"
    Target: "/data"
```

In this setup, the content available at the specified URL is downloaded and stored at the "/data" path within the task's environment. This mechanism ensures that tasks can directly access a broad range of web-based resources, augmenting the adaptability and utility of Bacalhau jobs.

### Example (Imperative/CLI)

When using the Bacalhau CLI to define the URL input source, you can employ the following imperative approach. Below are example commands demonstrating how to define the URL input source with various configurations:


1. **Fetch data from an HTTP endpoint and mount it**:
   This command demonstrates fetching data from a specific HTTP URL and mounting it to a designated path within the task's environment.
   ```bash
   bacalhau docker run -i http://example.com/data.txt ubuntu -- cat /input
   ```

2. **Fetch data from an HTTPS endpoint and mount it**:
   Similarly, you can fetch data from secure HTTPS URLs. This example fetches a file from a secure URL and mounts it.
   ```bash
   bacalhau docker run -i https://secure.example.com/data.txt:/data ubuntu -- cat /data
   ```
