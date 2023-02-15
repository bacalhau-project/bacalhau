---
sidebar_label: "DuckDB"
sidebar_position: 1
---
# DuckDB

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-engineering/DuckDB/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-engineering/DuckDB/index.ipynb)

## Introduction
DuckDB is a relational table-oriented database management system and supports SQL queries for producing analytical results. It also comes with various features that are useful for data analytics.

DuckDB is suited for the following use cases

Processing and storing tabular datasets, e.g. from CSV or Parquet files
Interactive data analysis, e.g. Joining & aggregate multiple large tables
Concurrent large changes, to multiple large tables, e.g. appending rows, adding/removing/updating columns
Large result set transfer to client

The advantage of using DuckDB with bacalhau is that you don’t need to install 
It locally also there is no need to download the datasets since the datasets are
Already there on IPFS or on the web


## Building Docker container

You can skip to running on bacalhau if you don’t want to build the container
If you want any additional dependencies to be installed along with DuckDB
you need to build your own container

To build your own docker container, create a Dockerfile, which contains 
Instructions to build your DuckDB   docker container


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

Building the container
```
docker build -t davidgasquez/datadex:v0.2.0 .
```

Testing it locally
```
❯ docker run davidgasquez/datadex:v0.1.0 "select 1"
┌───┐
│ 1 │
├───┤
│ 1 │
└───┘


```


Since our container is working locally we push it to docker hub
```
docker push davidgasquez/datadex:v0.2.0
```






## Running on bacalhau



```python
!curl -sL https://get.bacalhau.org/install.sh | bash
```

    Your system is linux_amd64
    No BACALHAU detected. Installing fresh BACALHAU CLI...
    Getting the latest BACALHAU CLI...
    Installing v0.3.11 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.3.11/bacalhau_v0.3.11_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.3.11/bacalhau_v0.3.11_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into /usr/local/bin successfully.
    Client Version: v0.3.11
    Server Version: v0.3.11


To test whether the same command that we ran locally runs on bacalhau we run the following command


```bash
%%bash --out job_id
bacalhau docker run \
     --workdir /inputs/ \
     --wait \
     --id-only \
     davidgasquez/datadex:v0.2.0 -- /bin/bash -c 'duckdb -s "select 1"'
```


```python
%env JOB_ID={job_id}
```

    env: JOB_ID=eb72c5f5-599b-464e-af93-3ecb9247e9af



```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 15:50:12 [0m[97;40m eb72c5f5 [0m[97;40m Docker davidgasquez/... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmXcsqrT1SvYZH... [0m


Where it says "Completed", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:



```bash
%%bash
bacalhau describe ${JOB_ID}
```

Downloading the outputs


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

    Fetching results of job 'eb72c5f5-599b-464e-af93-3ecb9247e9af'...
    Results for job 'eb72c5f5-599b-464e-af93-3ecb9247e9af' have been written to...
    results


    2022/11/11 15:52:13 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.


Viewing the outputs


```bash
%%bash
cat results/combined_results/stdout
```

    ┌───┐
    │ 1 │
    ├───┤
    │ 1 │
    └───┘


#wohooo! SQL on bacalhau


## Running Arbituary SQL commands over Yellow taxi trips dataset




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


```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
```

    [92;100m CREATED           [0m[92;100m ID                                   [0m[92;100m JOB                                                                                            [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED                                            [0m
    [97;40m 22-11-12-07:15:50 [0m[97;40m cced3685-2d50-4297-9739-6c692af8c60b [0m[97;40m Docker davidgasquez/duckdb:latest duckdb -s select count(*) from '0_yellow_taxi_trips.parquet' [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/Qmd3QYstyjEVkLrRRyEWVmhtEvmNMbjHcQ5a1o2zJy1JnJ [0m


Where it says "Completed", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:



```bash
%%bash
bacalhau describe ${JOB_ID}
```


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

    Fetching results of job 'cced3685-2d50-4297-9739-6c692af8c60b'...
    Results for job 'cced3685-2d50-4297-9739-6c692af8c60b' have been written to...
    results


    2022/11/12 07:19:32 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.


Viewing the outputs


```bash
%%bash
cat results/combined_results/stdout
```

    ┌──────────────┐
    │ count_star() │
    ├──────────────┤
    │ 24648499     │
    └──────────────┘

