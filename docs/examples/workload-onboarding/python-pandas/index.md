---
sidebar_label: "Python - Pandas"
sidebar_position: 2
---
# Running Pandas on Bacalhau

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/python-pandas/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/python-pandas/index.ipynb)

### Introduction

Pandas is a Python package that provides fast, flexible, and expressive data structures designed to make working with data both easy and intuitive. It aims to be the fundamental high-level building block for doing practical, real-world data analysis in Python. Additionally, it has the broader goal of becoming the most powerful and flexible open source data analysis/manipulation tool available in any language. It is already well on its way towards this goal.

### Prerequisites

* Python
* The Bacalhau client - [Installation instructions](https://docs.bacalhau.org/getting-started/installation)


## 1. Getting Started with Pandas Locally

The goal of this section is to show you how to develop a script to perform a task. We will then place this script in a container and run it at scale on Bacalhau. But first, you will need to install the Pandas library from pip.


```bash
%%bash
pip install pandas
```

### Importing data from CSV to DataFrame

Pandas is built around the idea of a DataFrame, a container for representing data. Below you will create a DataFrame by importing a CSV file. A CSV file is a text file with one record of data per line. The values within the record are separated using the “comma” character. Pandas provides a useful method, named `read_csv()` to read the contents of the CSV file into a DataFrame. For example, we can create a file named `transactions.csv` containing details of Transactions. The CSV file is stored in the same directory that contains Python script.



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

    hash,nonce,block_hash,block_number,transaction_index,from_address,to_address,value,gas,gas_price,input,block_timestamp,max_fee_per_gas,max_priority_fee_per_gas,transaction_type
    0x04cbcb236043d8fb7839e07bbc7f5eed692fb2ca55d897f1101eac3e3ad4fab8,12,0x246edb4b351d93c27926f4649bcf6c24366e2a7c7c718dc9158eea20c03bc6ae,483920,0,0x1b63142628311395ceafeea5667e7c9026c862ca,0xf4eced2f682ce333f96f2d8966c613ded8fc95dd,0,150853,50000000000,0xa9059cbb000000000000000000000000ac4df82fe37ea2187bc8c011a23d743b4f39019a00000000000000000000000000000000000000000000000000000000000186a0,1446561880,,,0
    0xcea6f89720cc1d2f46cc7a935463ae0b99dd5fad9c91bb7357de5421511cee49,84,0x246edb4b351d93c27926f4649bcf6c24366e2a7c7c718dc9158eea20c03bc6ae,483920,1,0x9b22a80d5c7b3374a05b446081f97d0a34079e7f,0xf4eced2f682ce333f96f2d8966c613ded8fc95dd,0,150853,50000000000,0xa9059cbb00000000000000000000000066f183060253cfbe45beff1e6e7ebbe318c81e560000000000000000000000000000000000000000000000000000000000030d40,1446561880,,,0
    0x463d53f0ad57677a3b430a007c1c31d15d62c37fab5eee598551697c297c235c,88,0x246edb4b351d93c27926f4649bcf6c24366e2a7c7c718dc9158eea20c03bc6ae,483920,2,0x9df428a91ff0f3635c8f0ce752933b9788926804,0x9e669f970ec0f49bb735f20799a7e7c4a1c274e2,11000440000000000,90000,50000000000,0x,1446561880,,,0
    0x05287a561f218418892ab053adfb3d919860988b19458c570c5c30f51c146f02,20085,0x246edb4b351d93c27926f4649bcf6c24366e2a7c7c718dc9158eea20c03bc6ae,483920,3,0x2a65aca4d5fc5b5c859090a6c34d164135398226,0x743b8aeedc163c0e3a0fe9f3910d146c48e70da8,1530219620000000000,90000,50000000000,0x,1446561880,,,0

### Running the script

Now let's run the script to read in the CSV file. The output will be a DataFrame object.


```bash
%%bash
python3 read_csv.py
```

                                                    hash  ...  transaction_type
    0  0x04cbcb236043d8fb7839e07bbc7f5eed692fb2ca55d8...  ...                 0
    1  0xcea6f89720cc1d2f46cc7a935463ae0b99dd5fad9c91...  ...                 0
    2  0x463d53f0ad57677a3b430a007c1c31d15d62c37fab5e...  ...                 0
    3  0x05287a561f218418892ab053adfb3d919860988b1945...  ...                 0
    
    [4 rows x 15 columns]


## 2. Running Pandas Jobs At Scale on Bacalhau

To run pandas on Bacalhau you must store your assets in a location that Bacalhau has access to. We usually default to storing data on IPFS and code in a container, but you can also easily upload your script to IPFS too.

If you are interested in finding out more about how to ingest your data into IPFS, please see the [data ingestion guide](../../data-ingestion/index.md).

We've already uploaded the script and data to IPFS to the following CID: `QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz`. You can look at this by browsing to one of the HTTP IPFS proxies like [ipfs.io](https://cloudflare-ipfs.com/ipfs/QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz/) or [w3s.link](https://bafybeih4hyydvojazlyv5zseelgn5u67iq2wbrbk2q4xoiw2d3cacdmzlu.ipfs.w3s.link/).

### Run the Job

Now we're ready to run a Bacalhau job, whilst mounting the Pandas script and data from IPFS. We'll use the `bacalhau docker run` command to do this. The `-v` flag allows us to mount a file or directory from IPFS into the container. The `-v` flag takes two arguments, the first is the IPFS CID and the second is the path to the directory in the container. The `-v` flag can be used multiple times to mount multiple directories.


```bash
%%bash --out job_id
 bacalhau docker run \
--wait \
--id-only \
-v QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz:/files \
-w /files \
amancevice/pandas \
-- python read_csv.py
```

Running the commands will output a UUID (like `e6377c99-b637-4661-a334-6ce98fcf037c`). This is the ID of the job that was created. You can check the status of the job with the following command:




```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 13:38:11 [0m[97;40m d48079d4 [0m[97;40m Docker amancevice/pa... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmY2MEETWyX77B... [0m



Where it says "`Published`", that means the job is done, and we can get the results.

If there is an error you can view the error using the following command bacalhau describe


```bash
%%bash
bacalhau describe ${JOB_ID}
```

    APIVersion: V1beta1
    ClientID: 77cf46c04f88ffb1c3e0e4b6e443724e8d2d87074d088ef1a6294a448fa85d2e
    CreatedAt: "2022-11-23T13:38:11.136995358Z"
    Deal:
      Concurrency: 1
    ExecutionPlan:
      ShardsTotal: 1
    ID: d48079d4-1358-4ce1-8a9e-5b9e6ae40bda
    JobState:
      Nodes:
        QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86:
          Shards:
            "0":
              NodeId: QmSyJ8VUd4YSPwZFJSJsHmmmmg7sd4BAc2yHY73nisJo86
              PublishedResults: {}
              State: Cancelled
              VerificationResult: {}
        QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF:
          Shards:
            "0":
              NodeId: QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF
              PublishedResults:
                CID: QmY2MEETWyX77BBYBNBpUW5bjkVAyP87EotPDVW2vjHG8K
                Name: job-d48079d4-1358-4ce1-8a9e-5b9e6ae40bda-shard-0-host-QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF
                StorageSource: IPFS
              RunOutput:
                exitCode: 0
                runnerError: ""
                stderr: ""
                stderrtruncated: false
                stdout: |2
                                                                  hash  ...  transaction_type
                  0  0x04cbcb236043d8fb7839e07bbc7f5eed692fb2ca55d8...  ...                 0
                  1  0xcea6f89720cc1d2f46cc7a935463ae0b99dd5fad9c91...  ...                 0
                  2  0x463d53f0ad57677a3b430a007c1c31d15d62c37fab5e...  ...                 0
                  3  0x05287a561f218418892ab053adfb3d919860988b1945...  ...                 0
    
                  [4 rows x 15 columns]
                stdouttruncated: false
              State: Completed
              Status: 'Got results proposal of length: 0'
              VerificationResult:
                Complete: true
                Result: true
    RequesterNodeID: QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF
    RequesterPublicKey: CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCehDIWl72XKJi1tsrYM9JjAWt3n6hNzrCA+IVRXixK1sJVTLMpsxEP8UKJI+koAWkAUuY8yi6DMzot0owK4VpM3PYp34HdKi2hTjzM8pjCVb70XVXt6k9bzj4KmbiQTuEkQfvwIRmgxb2jrkRdTpZmhMb1Q7StR/nrGa/bx75Vpupx1EYH6+LixYnnV5WbCUK/kjpBW8SF5v+f9ZO61KHd9DMpdhJnzocTGq17tAjHh3birke0xlP98JjxlMkzzvIAuFsnH0zBIgjmHDA1Yi5DcOPWgE0jUfGlSDC1t2xITVoofHQcXDjkHZE6OhxswNYPd7cnTf9OppLddFdQnga5AgMBAAE=
    Spec:
      Docker:
        Entrypoint:
        - python
        - read_csv.py
        Image: amancevice/pandas
        WorkingDirectory: /files
      Engine: Docker
      Language:
        JobContext: {}
      Publisher: Estuary
      Resources:
        GPU: ""
      Sharding:
        BatchSize: 1
        GlobPatternBasePath: /inputs
      Timeout: 1800
      Verifier: Noop
      Wasm: {}
      inputs:
      - CID: QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz
        StorageSource: IPFS
        path: /files
      outputs:
      - Name: outputs
        StorageSource: IPFS
        path: /outputs


The describe command will display the logs and error messages from your job. There's no errors this time (lucky?) so now let's create a temporary directory to save our results.


```bash
%%bash
rm -rf results && mkdir -p results
```

To Download the results of your job, run the following command:


```bash
%%bash
bacalhau get ${JOB_ID}  --output-dir results
```

    Fetching results of job 'd48079d4-1358-4ce1-8a9e-5b9e6ae40bda'...
    Results for job 'd48079d4-1358-4ce1-8a9e-5b9e6ae40bda' have been written to...
    results


After the download has finished you should 
see the following contents in pandas-results directory


```bash
%%bash
ls results/combined_results/
```

    outputs
    stderr
    stdout


The structure of the files and directories will look like this:

```
.
├── combined_results
│   ├── outputs
│   ├── stderr
│   └── stdout
├── per_shard
│   └── 0_node_QmSyJ8VU
│       ├── exitCode
│       ├── outputs
│       ├── stderr
│       └── stdout
└── raw
    └── QmY2MEETWyX77BBYBNBpUW5bjkVAyP87EotPDVW2vjHG8K
        ├── exitCode
        ├── outputs
        ├── stderr
        └── stdout
```

* `stdout` contains things printed to the console like outputs, etc.

* `stderr` contains any errors. In this case, since there are no errors, it's will be empty

* `outputs` folder is the volume you named when you started the job with the `-o` flag. In addition, you will always have a `outputs` volume, which is provided by default.

Because your script is printed to stdout, the output will appear in the stdout file. You can read this by typing the following command:





```bash
%%bash
cat results/combined_results/stdout
```

                                                    hash  ...  transaction_type
    0  0x04cbcb236043d8fb7839e07bbc7f5eed692fb2ca55d8...  ...                 0
    1  0xcea6f89720cc1d2f46cc7a935463ae0b99dd5fad9c91...  ...                 0
    2  0x463d53f0ad57677a3b430a007c1c31d15d62c37fab5e...  ...                 0
    3  0x05287a561f218418892ab053adfb3d919860988b1945...  ...                 0
    
    [4 rows x 15 columns]


Success! The next step is to scale up your data and your processing via multiple jobs or sharding. You might be interested in looking at:

* [An example running hundreds of jobs over "big data"](../../data-engineering/blockchain-etl/index.md)
* [A simple sharding example](../../data-engineering/simple-parallel-workloads/index.md)
