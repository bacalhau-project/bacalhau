# Flyte Bacalhau Plugin

This repo adheres to the [Flyte official guidelines](https://github.com/flyteorg/flytekit/tree/master/plugins#guidelines-) for flytekit plugins and is structured such that the `plugins/flytekit-bacalhau` folder can be moved into Flytekit repository.

Author: @enricorotundo

## Development :computer:

Similarly to the [official development guidelines](https://docs.flyte.org/projects/flytekit/en/latest/contributing.html#contribute-code), we use a virtual environment to develop this Flytekit plugin.

### 1. Setup (Do Once)

```bash
virtualenv ~/.virtualenvs/flytekit-bacalhau
source ~/.virtualenvs/flytekit-bacalhau/bin/activate
make setup
```

> It is important to maintain separate virtualenvs for `flytekit development` and `flytekit` use because installing a Python library in editable mode will link it to your source code. The behavior will change as you work on the code, check out different branches, etc.

This will install Flytekit dependencies and Flytekit in editable mode. This links your virtual Python’s site-packages with your local repo folder, allowing your local changes to take effect when the same Python interpreter runs import flytekit.


### 2. Plugin Development

```bash
source ~/.virtualenvs/flytekit-bacalhau/bin/activate
cd plugins
pip install -e .
```

This should install all the plugins in editable mode as well.

#### Unit tests

```bash
make test
```

### Formatting, Linting, etc.

```bash
source ~/.virtualenvs/flytekit/bin/activate
make fmt
make lint
make spellcheck
```
