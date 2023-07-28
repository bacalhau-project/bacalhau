# Flyte Bacalhau Plugin

## Repo structure

TODO - expand on why this struct...

This repo adheres to the [Flyte official guidelines](https://github.com/flyteorg/flytekit/tree/master/plugins#guidelines-) for flytekit plugins.

## Development :computer:

### 1. Setup env

Similar to https://docs.flyte.org/projects/flytekit/en/latest/contributing.html#contribute-code

```bash
virtualenv ~/.virtualenvs/flytekit-bacalhau
source ~/.virtualenvs/flytekit-bacalhau/bin/activate
make setup
```

> It is important to maintain separate virtualenvs for flytekit development and flytekit use because installing a Python library in editable mode will link it to your source code. The behavior will change as you work on the code, check out different branches, etc.

This will install Flytekit dependencies and Flytekit in editable mode. This links your virtual Python’s site-packages with your local repo folder, allowing your local changes to take effect when the same Python interpreter runs import flytekit.


### 2. Plugin dev env

```bash
source ~/.virtualenvs/flytekit-bacalhau/bin/activate
cd plugins
pip install -e .
```

This should install all the plugins in editable mode as well.

### 3. Pre-commit hooks

TODO
https://docs.flyte.org/projects/flytekit/en/latest/contributing.html#pre-commit-hooks
### 4. Formatting

TODO
https://docs.flyte.org/projects/flytekit/en/latest/contributing.html#pre-commit-hooks

--- 



### Unit tests

```bash
make test
```
