
# Flytekit Bacalhau Plugin

Bacalhau is a platform for fast, cost efficient, and secure computation by running jobs where the data is generated and stored. With Bacalhau you can streamline your existing workflows without the need of extensive rewriting by running arbitrary Docker containers and WebAssembly (wasm) images as tasks.

To install the plugin, run the following command:

```bash
$ pip install flytekitplugins-bacalhau
```

## Task Example
```python
$ python flytekit-bacalhau/scripts/wf.py
# or
$ pyflyte run flytekit-bacalhau/scripts/wf.py wf
```

More examples can be found in the documentation. - TODO(@enricorotundo)

<!-- ## Describe Agent? -->

