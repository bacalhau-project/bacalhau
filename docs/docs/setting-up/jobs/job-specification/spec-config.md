---
sidebar_label: SpecConfig
---

# SpecConfig Specification

`SpecConfig` provides a unified structure to specify configurations for various components in Bacalhau, including engines, publishers, and input sources. Its flexible design allows seamless integration with multiple systems like Docker, WebAssembly (Wasm), AWS S3, and local directories, among others.


## `SpecConfig` Parameters

- **Type** `(string : <required>)`: Specifies the type of the configuration. Examples include `docker` and `wasm` for execution engines, `S3` for input sources and publishers, etc.

- **Params** `(map[string]any : <optional>)`: A set of key-value pairs that provide the specific configurations for the chosen type. The keys and values are flexible and depend on the `Type`. For instance, parameters for a Docker engine might include image name and version, while an S3 publisher would require configurations like the bucket name and AWS region. If not provided, it defaults to `nil`.

## Usage Examples

Here are a few hypothetical examples to demonstrate how you might define `SpecConfig` for different components:

### Docker Engine

```json
{
  "Type": "docker",
  "Params": {
    "Image": "my_app_image",
    "Entrypoint": "my_app_entrypoint",
  }
}
```
Full Docker spec can be found [here](../../other-specifications/engines/docker).

### S3 Publisher

```json
{
  "Type": "s3",
  "Params": {
    "Bucket": "my_bucket",
    "Region": "us-west-1"
  }
}
```
Full S3 Publisher can be found [here](../../other-specifications/publishers/s3).

### Local Directory Input Source

```json
{
  "Type": "localDirectory",
  "Params": {
    "SourcePath": "/path/to/local/directory",
    "ReadWrite": true,
  }
}
```
Full local source can be found [here](../../other-specifications/sources/local.md).

Remember, the exact keys and values in the `Params` map will vary depending on the specific requirements of the component being configured. Always refer to the individual component's documentation to understand the available parameters.
