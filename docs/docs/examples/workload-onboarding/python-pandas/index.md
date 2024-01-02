---
sidebar_label: "Python Pandas"
sidebar_position: 6
---
# Running Pandas on Bacalhau


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

### Introduction

Pandas is a Python package that provides fast, flexible, and expressive data structures designed to make working with data both easy and intuitive. It aims to be the fundamental high-level building block for doing practical, real-world data analysis in Python. Additionally, it has the broader goal of becoming the most powerful and flexible open-source data analysis/manipulation tool available in any language. It is already well on its way towards this goal.

## TD;LR
Running pandas script in Bacalhau

## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)


## Running Pandas Locally

To run the Pandas script on Bacalhau for analysis, first, we will place the Pandas script in a container and then run it at scale on Bacalhau. To get started, you need to install the Pandas library from pip.


```bash
%%bash
pip install pandas
```

### Importing data from CSV to DataFrame

Pandas is built around the idea of a DataFrame, a container for representing data. Below you will create a DataFrame by importing a CSV file. A CSV file is a text file with one record of data per line. The values within the record are separated using the “comma” character. Pandas provides a useful method, named `read_csv()` to read the contents of the CSV file into a DataFrame. For example, we can create a file named `transactions.csv` containing details of Transactions. The CSV file is stored in the same directory that contains the Python script.



```python
%%writefile read_csv.py
import pandas as pd

print(pd.read_csv("transactions.csv"))
```


```bash
%%bash
# Downloading the dataset
wget https://cloudflare-ipfs.com/ipfs/QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz/transactions.csv
```


```bash
%%bash
cat transactions.csv
```

### Running the script

Now let's run the script to read in the CSV file. The output will be a DataFrame object.


```bash
%%bash
python3 read_csv.py
```

## Ingesting data

To run pandas on Bacalhau you must store your assets in a location that Bacalhau has access to. We usually default to storing data on IPFS and code in a container, but you can also easily upload your script to IPFS too.

If you are interested in finding out more about how to ingest your data into IPFS, please see the [data ingestion guide](https://docs.bacalhau.org/examples/data-ingestion/).

We've already uploaded the script and data to IPFS to the following CID: `QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz`. You can look at this by browsing to one of the HTTP IPFS proxies like [ipfs.io](https://cloudflare-ipfs.com/ipfs/QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz/) or [w3s.link](https://bafybeih4hyydvojazlyv5zseelgn5u67iq2wbrbk2q4xoiw2d3cacdmzlu.ipfs.w3s.link/).

## Running a Bacalhau Job

After mounting the Pandas script and data from IPFS, we can now use the container for running on Bacalhau. To submit a job, run the following Bacalhau command:

Now we're ready to run a Bacalhau job, whilst mounting the Pandas script and data from IPFS. We'll use the `bacalhau docker run` command to do this. The `-v` flag allows us to mount a file or directory from IPFS into the container. The `-v` flag takes two arguments, the first is the IPFS CID and the second is the path to the directory in the container. The `-v` flag can be used multiple times to mount multiple directories.


```bash
%%bash --out job_id
bacalhau docker run \
    --wait \
    --id-only \
    -i ipfs://QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz:/files \
    -w /files \
    amancevice/pandas \
    -- python read_csv.py
```

### Structure of the command

- `bacalhau docker run`: call to bacalhau

- `amancevice/pandas `: Using the official pytorch Docker image

- ``-i ipfs://QmfKJT13h5k1b23ja3Z .....`: Mounting the uploaded dataset to path

- `-w /files` Our working directory is /outputs. This is the folder where we will save the model as it will automatically get uploaded to IPFS as outputs

` python read_csv.py`: python script to read pandas script

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

When it says `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe ${JOB_ID}
```

When it says `Published` or `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get ${JOB_ID}  --output-dir results
```

## Viewing your Job Output

To view the file, run the following command:


```bash
%%bash
cat results/stdout # displays the contents of the file
```
