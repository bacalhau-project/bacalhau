# Configuration Flag Usage

The `--config` (or `-c`) flag allows flexible configuration of the application through various methods. You can use this flag multiple times to combine different configuration sources.

## Usage

```
bacalhau [command] --config <option> [--config <option> ...]
```

or using the short form:

```
bacalhau [command] -c <option> [-c <option> ...]
```

## Configuration Options

1. **YAML Config Files**: Specify paths to YAML configuration files.
   Example: `-c path/to/config.yaml`

2. **Key-Value Pairs**: Set specific configuration values using dot notation.
   Example: `-c WebUI.Enabled=true`

3. **Boolean Flags**: Enable boolean options by specifying the key alone.
   Example: `-c WebUI.Enabled`

## Precedence

When multiple configuration options are provided, they are applied in the following order of precedence (highest to lowest):

1. Command-line key-value pairs and boolean flags
2. YAML configuration files
3. Default values

Within each category, options specified later override earlier ones.

## Examples

1. Using a single config file:
   ```
   bacalhau serve --config my-config.yaml
   ```

2. Merging multiple config files:
   ```
   bacalhau serve -c base-config.yaml -c override-config.yaml
   ```

3. Overriding specific values:
   ```
   bacalhau serve -c config.yaml -c WebUI.Port=9090 -c Node.Name=custom-node
   ```

4. Combining file and multiple overrides:
   ```
   bacalhau serve -c config.yaml -c WebUI.Enabled -c Node.ClientAPI.Host=192.168.1.5
   ```

In the last example, `WebUI.Enabled` will be set to `true`, `Node.ClientAPI.Host` will be "192.168.1.5", and other values will be loaded from `config.yaml` if present.

Remember, later options override earlier ones, allowing for flexible configuration management.