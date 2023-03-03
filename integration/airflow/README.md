# Apache Airflow Provider for Bacalhau

This is the `bacalhau-airflow`, a python package including an Apache Airflow provider.

What's a provider???

## Features

- Create Airflow tasks that run on Bacalhau (via custom operator!)
- Support for sharded jobs: output shards can be passed downstream (via XComs)
- Coming soon...
    - Lineage (OpenLineage)
    - Various working code examples
    - Hosting instructions

## Requirements

- Python 3.8+
- `bacalhau-sdk` 0.1.5
- `apache-airflow` 2.3+

## Installation

## From pypi

```console
pip install bacalhau-airflow
```

## From source

Clone the public repository:

```shell
git clone https://github.com/bacalhau-project/bacalhau/
```

Once you have a copy of the source, you can install it with:

```shell
cd integration/airflow/
pip install .
```

## Setup

The integration automatically registers itself for Airflow 2.3 if it's installed on the Airflow worker's Python.

## Use

First, install and initalize Airflow:

```shell
pip install apache-airflow
export AIRFLOW_HOME=~/airflow
airflow db init
```

Then, we need to point Airflow to the absolute path of the folder where your pipelines live.
To do that we edit the `dags_folder` field in `${AIRFLOW_HOME}/airflow.cfg` file.
Optionally, to reduce clutter in the Airflow UI, you could also set `load_examples` to `False`.

We can

```shell
airflow standalone
```


## Development


```shell
pip install -r dev-requirements.txt
```

### Unit tests


```shell
tox
```

You can also skip using `tox` and run `pytest` on your own dev environment.
