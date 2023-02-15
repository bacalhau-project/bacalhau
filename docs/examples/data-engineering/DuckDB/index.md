---
sidebar_label: "DuckDB"
sidebar_position: 1
---
# Using Bacalhau with DuckDB

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-engineering/DuckDB/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-engineering/DuckDB/index.ipynb)


DuckDB is a relational table-oriented database management system and supports SQL queries for producing analytical results. It also comes with various features that are useful for data analytics.

DuckDB is suited for the following use cases:

- Processing and storing tabular datasets, e.g. from CSV or Parquet files
- Interactive data analysis, e.g. Joining & aggregate multiple large tables
- Concurrent large changes, to multiple large tables, e.g. appending rows, adding/removing/updating columns
- Large result set transfer to client

In this example tutorial, we will show how to use DuckDB with Bacalhau. The advantage of using DuckDB with Bacalhau is that you don’t need to install,  there is no need to download the datasets since the datasets are
already there on IPFS or on the web.


## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Building Docker container

:::Info
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

### Building the container

We will run `docker build` command to build the container;

```
docker build -t <hub-user>/<repo-name>:<tag>
```

Before running the command replace;

- **hub-user** with your docker hub username, If you don’t have a docker hub account [Follow these instructions to create docker account](https://docs.docker.com/docker-id/), and use the username of the account you created

- **repo-name** with the name of the container, you can name it anything you want

- **tag** this is not required but you can use the latest tag

After you have build the container, the next step is to test it locally and then push it docker hub. Before pushing you first need to create a repo which you can create by following the instructions here [https://docs.docker.com/docker-hub/repos/](https://docs.docker.com/docker-hub/repos/)


Now you can push this repository to the registry designated by its name or tag.

```
 docker push <hub-user>/<repo-name>:<tag>
```

## Running a Bacalhau Job

To run the container on Bacalhau, we will use the `bacalhau docker run` command.


```bash
%%bash --out job_id
bacalhau docker run \
     --workdir /inputs/ \
     --wait \
     --id-only \
     davidgasquez/datadex:v0.2.0 -- /bin/bash -c 'duckdb -s "select 1"'
```

When a job is sumbitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%%env JOB_ID={job_id}
```

    env: JOB_ID=eb72c5f5-599b-464e-af93-3ecb9247e9af


## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`. 


```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 15:50:12 [0m[97;40m eb72c5f5 [0m[97;40m Docker davidgasquez/... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmXcsqrT1SvYZH... [0m


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

    Fetching results of job 'eb72c5f5-599b-464e-af93-3ecb9247e9af'...
    Results for job 'eb72c5f5-599b-464e-af93-3ecb9247e9af' have been written to...
    results


    2022/11/11 15:52:13 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.


## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**,**per_shard files**, and the **raw** directory. To view the file, run the following command:


```bash
%%bash
cat results/combined_results/stdout
```

    ┌───┐
    │ 1 │
    ├───┤
    │ 1 │
    └───┘


## Running Arbituary SQL commands

Below is the `bacalhau docker run` command to to run arbituary SQL commands over yellow taxi trips dataset


```bash
%%bash --out job_id
bacalhau docker run \
 -i bafybeiejgmdpwlfgo3dzfxfv3cn55qgnxmghyv7vcarqe3onmtzczohwaq \
  --workdir /inputs \
  --id-only \
  --wait \
  davidgasquez/duckdb:latest \
  -- duckdb -s "select count(*) from '0_yellow_taxi_trips.parquet'"

```

When a job is sumbitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

- **Job status**: You can check the status of the job using `bacalhau list`. 


```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
```

    [92;100m CREATED           [0m[92;100m ID                                   [0m[92;100m JOB                                                                                            [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED                                            [0m
    [97;40m 22-11-12-07:15:50 [0m[97;40m cced3685-2d50-4297-9739-6c692af8c60b [0m[97;40m Docker davidgasquez/duckdb:latest duckdb -s select count(*) from '0_yellow_taxi_trips.parquet' [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/Qmd3QYstyjEVkLrRRyEWVmhtEvmNMbjHcQ5a1o2zJy1JnJ [0m


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

    Fetching results of job 'cced3685-2d50-4297-9739-6c692af8c60b'...
    Results for job 'cced3685-2d50-4297-9739-6c692af8c60b' have been written to...
    results


    2022/11/12 07:19:32 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.


## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**,**per_shard files**, and the **raw** directory. To view the file, run the following command:


```bash
%%bash
cat results/combined_results/stdout
```

    ┌──────────────┐
    │ count_star() │
    ├──────────────┤
    │ 24648499     │
    └──────────────┘

