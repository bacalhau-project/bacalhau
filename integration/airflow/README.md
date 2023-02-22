# Airflow Provider for Bacalhau

## Features

- Create Airflow tasks that run on Bacalhau (via custom operator)
- Support for sharded jobs: output shards can be passed downstream via XCom
- Lineage (Coming soon...)
- more (Coming soon...)
## Requirements

- Python 3.8+
- bacalhau-sdk
- apache-airflow +2.4

## Installation

Install from source:

```shell
pip install .
```

This package is named `bacalhau`.

## Usage (WIP :warning:)

```
AIRFLOW_VERSION=2.4.1
PYTHON_VERSION="$(python --version | cut -d " " -f 2 | cut -d "." -f 1-2)"
CONSTRAINT_URL="https://raw.githubusercontent.com/apache/airflow/constraints-${AIRFLOW_VERSION}/constraints-${PYTHON_VERSION}.txt"
pip install "apache-airflow==${AIRFLOW_VERSION}" --constraint "${CONSTRAINT_URL}"

export AIRFLOW_HOME=~/airflow
airflow db init
```

```
export AIRFLOW_HOME=~/airflow
airflow standalone
```


## Development


```shell
$ pip install -r dev-requirements.txt
```

### Unit tests


```shell
tox
```

You can also skip using `tox` and run `pytest` on your own dev environment.

