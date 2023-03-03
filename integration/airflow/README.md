# Apache Airflow Provider for Bacalhau

This is `bacalhau-airflow`, a Python package that integrates Bacalhau with Apache Airflow.
The benefit is two fold.
First, thanks to this package you can now write complex pipelines for Bacalhau. For instance, jobs can communicate their output's CIDs to downstream jobs, that can use those as inputs.
Second, Apache Airflow provides a solid solution to reliably orchestrate your DAGs.

## Features

- Create Airflow tasks that run on Bacalhau (via custom operator!)
- Support for sharded jobs: output shards can be passed downstream (via XComs TODO link to xcoms)
- Coming soon...
    - Lineage (via OpenLineage)
    - Various working code examples
    - Hosting instructions

## Requirements

- Python 3.8+
- `bacalhau-sdk` 0.1.6 TODO add link to pypi
- `apache-airflow` 2.3+

## Installation

The integration automatically registers itself for Airflow 2.3+ if it's installed on the Airflow worker's Python.

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

For a production environment you may want to deploy Airflow in one of the many official / suggested options.

If you're just curious and want to give it a try on your local machine, please follow the steps below.

First, install and initalize Airflow:

```shell
pip install apache-airflow
export AIRFLOW_HOME=~/airflow
airflow db init
```

Then, we need to point Airflow to the absolute path of the folder where your pipelines live.
To do that we edit the `dags_folder` field in `${AIRFLOW_HOME}/airflow.cfg` file.
In this example I'm going to use the `hello_world.py` DAG shipped with this repository.
My config file looks like what follows:

```
[core]
dags_folder = /Users/enricorotundo/bacalhau/integration/airflow/example_dags
...
```

*Optionally, to reduce clutter in the Airflow UI, you could disable the loading of the default example DAGs by setting `load_examples` to `False`.*

Finally, we can launch Airflow locally:

```shell
airflow standalone
```

Now head to http://0.0.0.0:8080 were Airflow UI is being served.
The screenshot below shows our hello world has been loaded correctly.

![](docs/_static/airflow_1.png)

When you inspect a DAG, Airflow will render a graph depicting a color-coded topology (see image below).
For active (i.e. running) pipelines, this will be useful to oversee what the status of each task is.
To trigger a DAG please enable the toggle shown below.

![](docs/_static/airflow_2.png)

Lastly, we want to fetch the output of our pipeline.
To do so we need to retrieve where the last task saved its results.
TODO...

## Development


```shell
pip install -r dev-requirements.txt
```

### Unit tests


```shell
tox
```

You can also skip using `tox` and run `pytest` on your own dev environment.
