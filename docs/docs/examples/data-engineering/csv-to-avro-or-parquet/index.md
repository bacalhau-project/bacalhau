---
sidebar_label: "csv-to-avro-or-parquet"
sidebar_position: 2
---
# Convert CSV To Parquet Or Avro


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

## Introduction

Converting from CSV to parquet or avro reduces the size of the file and allows for faster read and write speeds. With Bacalhau, you can convert your CSV files stored on ipfs or on the web without the need to download files and install dependencies locally.

In this example tutorial we will convert a CSV file from a URL to parquet format and save the converted parquet file to IPFS

## TD;LR
Converting CSV stored in public storage with Bacalhau


## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)
```
!command -v bacalhau >/dev/null 2>&1 || (export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
path=!echo $PATH
%env PATH=./:{path[0]}
```

## Running CSV to Arvo or Parquet Locally​


Installing dependencies



```python
%cd csv_to_avro_or_parquet
```

## Install the following dependencies

Run the following commands:


```bash

%%bash
pip install fastavro
```


```bash

%%bash
pip install numpy
```


```bash

%%bash
pip install pandas
```


```bash
%%bash
pip install pyarrow
```


```bash
%%bash
python3 src/converter.py ./movies.csv  ./movies.parquet parquet

# python converter.py path_to_csv path_to_result_file extension
```

Viewing the parquet file


```python
import pandas as pd
pd.read_parquet('./movies.parquet').head()
```

## Containerize Script with Docker

:::info
You can skip this section entirely and directly go to running on Bacalhau
:::

To build your own docker container, create a `Dockerfile`, which contains instructions to build your image.

```
FROM python:3.8

RUN apt update && apt install git

RUN git clone https://github.com/bacalhau-project/Sparkov_Data_Generation

WORKDIR /Sparkov_Data_Generation/

RUN pip3 install -r requirements.txt
```

:::info
See more information on how to containerize your script/app[here](https://docs.docker.com/get-started/02_our_app/)
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

In our case:

```
docker build -t jsacex/csv-to-arrow-or-parquet
```

### Push the container

Next, upload the image to the registry. This can be done by using the Docker hub username, repo name or tag.

```
docker push <hub-user>/<repo-name>:<tag>
```

In our case:

```
docker push jsacex/csv-to-arrow-or-parquet
```

## Running a Bacalhau Job

To submit a job, we are going to either mount the script from an IPFS or from an URL.

### Mounting the CSV File from IPFS

With the command below, we are gmounting the CSV file for transactions from IPFS


```python
!command -v bacalhau >/dev/null 2>&1 || (export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
path=!echo $PATH
%env PATH=./:{path[0]}
```


```bash
%%bash --out job_id
bacalhau docker run \
-i ipfs://QmTAQMGiSv9xocaB4PUCT5nSBHrf9HZrYj21BAZ5nMTY2W  \
--wait \
--id-only \
jsacex/csv-to-arrow-or-parquet \
-- python3 src/converter.py ../inputs/transactions.csv  ../outputs/transactions.parquet parquet
```

### Structure of the command

Let's look closely at the command above:

* `bacalhau docker run`: call to bacalhau

* `-i ipfs://QmTAQMGiSv9xocaB4PUCT5nSBHrf9HZrYj21BAZ5nMTY2W`: CIDs to use on the job. Mounts them at '/inputs' in the execution.

* `jsacex/csv-to-arrow-or-parque`: the name and the tag of the docker image we are using

* `../inputs/movies.csv `: path to input dataset

* `../outputs/movies.parquet parquet`: path to the output

* `python3 src/converter.py`: execute the script

### Mounting the CSV File from a URL
To mount the CSV file from a URL


```bash
%%bash --out job_id
bacalhau docker run \
-i https://raw.githubusercontent.com/bacalhau-project/csv_to_avro_or_parquet/master/movies.csv \
jsacex/csv-to-arrow-or-parquet \
-- python3 src/converter.py ../inputs/movies.csv  ../outputs/movies.parquet parquet
```

### Structure of the command

Let's look closely at the command above:

* `bacalhau docker run`: call to bacalhau

* `-i https://raw.githubusercontent.com/bacalhau-project/csv_to_avro_or_parquet/master/movies.csv`: URL: path of the input data volumes downloaded from a URL source

* `jsacex/csv-to-arrow-or-parque`: the name and the tag of the docker image we are using

* `../inputs/movies.csv `: path to the input dataset

* `../outputs/movies.parquet parquet`: path to the output

* `python3 src/converter.py`: execute the script

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%env JOB_ID={job_id}
```

    env: JOB_ID=bacalhau describe 71ecde0e-dac3-4c8d-bf2e-7a92cc54425e


## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`.

:::note
Replace the `{JOB_ID}` with your generated ID.
:::


```bash
%%bash
bacalhau list --id-filter={JOB_ID}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 16:53:30 [0m[97;40m 71ecde0e [0m[97;40m Docker jsacex/csv-to... [0m[97;40m Completed [0m[97;40m          [0m[97;40m ipfs://QmP5PbbJZ1fdq... [0m



When it says `Published` or `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe {JOB_ID}
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get ${JOB_ID} --output-dir results
```

## Viewing your Job Output

To view the file, run the following command:


```bash
%%bash
ls results/outputs
```

Alternatively, you can do this.


```python
import pandas as pd
import os
pd.read_parquet('results/outputs/transactions.parquet')
```

## Need Support?

For questions, and feedback, please reach out in our [forum](https://github.com/filecoin-project/bacalhau/discussions)
