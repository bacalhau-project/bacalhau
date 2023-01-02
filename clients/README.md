# Push new `bacalhau-apiclient` version

1. Bump version in `pyproject.toml`
1. `make swagger-docs clients` (from repo root)
1. `cd clients && make pypi-upload` (requires Pypi.org credentials)
1. Check the new version at https://pypi.org/project/bacalhau-apiclient/#history
