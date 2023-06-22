---
sidebar_label: "DuckDB"
sidebar_position: 3
---
# Using Bacalhau with DuckDB


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)


DuckDB is a relational table-oriented database management system that supports SQL queries for producing analytical results. It also comes with various features that are useful for data analytics.

DuckDB is suited for the following use cases:

- Processing and storing tabular datasets, e.g. from CSV or Parquet files
- Interactive data analysis, e.g. Joining & aggregate multiple large tables
- Concurrent large changes, to multiple large tables, e.g. appending rows, adding/removing/updating columns
- Large result set transfer to client

In this example tutorial, we will show how to use DuckDB with Bacalhau. The advantage of using DuckDB with Bacalhau is that you don’t need to install, and there is no need to download the datasets since the datasets are
already there on IPFS or on the web.

## TD;lR
Running a relational database(DUCKDB) on Bacalhau

## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Containerize Script using Docker

:::info
You can skip this entirely and directly go to running on Bacalhau.
:::

If you want any additional dependencies to be installed along with DuckDB, you need to build your own container.

To build your own docker container, create a `Dockerfile`, which contains instructions to build your DuckDB docker container.


```Dockerfile
FROM mcr.microsoft.com/vscode/devcontainers/python:3.9

RUN apt-get update && apt-get install -y nodejs npm g++

# Install dbt
RUN pip3 --disable-pip-version-check --no-cache-dir install duckdb==0.4.0 dbt-duckdb==1.1.4 \
    && rm -rf /tmp/pip-tmp

# Install duckdb cli
RUN wget https://github.com/duckdb/duckdb/releases/download/v0.4.0/duckdb_cli-linux-amd64.zip \
    && unzip duckdb_cli-linux-amd64.zip -d /usr/local/bin \
    && rm duckdb_cli-linux-amd64.zip

# Configure Workspace
ENV DBT_PROFILES_DIR=/workspaces/datadex
WORKDIR /workspaces/datadex

```

:::info
See more information on how to containerize your script/app [here](https://docs.docker.com/get-started/02_our_app/)
:::


### Build the container

We will run `docker build` command to build the container;

```
docker build -t <hub-user>/<repo-name>:<tag> .
```

Before running the command replace;

- **hub-user** with your docker hub username, If you don’t have a docker hub account [follow these instructions to create docker account](https://docs.docker.com/docker-id/), and use the username of the account you created

- **repo-name** with the name of the container, you can name it anything you want

- **tag** this is not required but you can use the latest tag

In our case

```bash
docker build -t davidgasquez/datadex:v0.2.0
```

### Push the container

Next, upload the image to the registry. This can be done by using the Docker hub username, repo name or tag.

```
docker push <hub-user>/<repo-name>:<tag>
```

In our case

```bash
docker push davidgasquez/datadex:v0.2.0
```

## Running a Bacalhau Job

After the repo image has been pushed to Docker Hub, we can now use the container for running on Bacalhau. To submit a job, run the following Bacalhau command:


```bash
%%bash --out job_id
bacalhau docker run \
--workdir /inputs/ \
--wait \
--id-only \
davidgasquez/datadex:v0.2.0 -- /bin/bash -c 'duckdb -s "select 1"'
```

### Structure of the command

Let's look closely at the command above:

* `bacalhau docker run`: call to bacalhau

* `davidgasquez/datadex:v0.2.0 `: the name and the tag of the docker image we are using

* `/inputs/`: path to input dataset

* `'duckdb -s "select 1"'`: execute DuckDB


When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%env JOB_ID={job_id}
```

## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

When it says `Published` or `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.



```bash
%%bash
bacalhau describe ${JOB_ID}
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**,**per_shard files**, and the **raw** directory. To view the file, run the following command:


```bash
%%bash
cat results/stdout  # displays the contents of the file
```

    ┌───┐
    │ 1 │
    ├───┤
    │ 1 │
    └───┘


## Running Arbitrary SQL commands

Below is the `bacalhau docker run` command to to run arbitrary SQL commands over the yellow taxi trips dataset


```bash
%%bash --out job_id
bacalhau docker run \
 -i ipfs://bafybeiejgmdpwlfgo3dzfxfv3cn55qgnxmghyv7vcarqe3onmtzczohwaq \
  --workdir /inputs \
  --id-only \
  --wait \
  davidgasquez/duckdb:latest \
  -- duckdb -s "select count(*) from '0_yellow_taxi_trips.parquet'"

```

### Structure of the command

Let's look closely at the command above:

* `bacalhau docker run`: call to bacalhau

* `-i ipfs://bafybeiejgmdpwlfgo3dzfxfv3cn55qgnxmghyv7vcarqe3onmtzczohwaq \`: CIDs to use on the job. Mounts them at '/inputs' in the execution.

* `davidgasquez/duckdb:latest`: the name and the tag of the docker image we are using

* `/inputs`: path to input dataset

* `duckdb -s`: execute DuckDB


When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

- **Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
```

- **Job information**: You can find out more information about your job by using `bacalhau describe`.



```bash
%%bash
bacalhau describe ${JOB_ID}
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**,**per_shard files**, and the **raw** directory. To view the file, run the following command:


```bash
%%bash
cat results/stdout
```

    ┌──────────────┐
    │ count_star() │
    │    int64     │
    ├──────────────┤
    │     24648499 │
    └──────────────┘


## Need Support?

For questions, and feedback, please reach out in our [forum](https://github.com/filecoin-project/bacalhau/discussions)
