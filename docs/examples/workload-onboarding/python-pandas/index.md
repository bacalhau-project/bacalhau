---
sidebar_label: "Python Pandas"
sidebar_position: 6
---
# Running Pandas on Bacalhau


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

### Introduction

Pandas is a Python package that provides fast, flexible, and expressive data structures designed to make working with data both easy and intuitive. It aims to be the fundamental high-level building block for doing practical, real-world data analysis in Python. Additionally, it has the broader goal of becoming the most powerful and flexible open source data analysis/manipulation tool available in any language. It is already well on its way towards this goal.

## TD;LR
Running pandas script in Bacalhau

## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)


## Running Pandas Locally

To run Pandas script on Bacalhau for analysis, first we will place the Pandas script in a container and then run it at scale on Bacalhau. To get started, you need to install the Pandas library from pip.


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

- `-w /files` Our working directory is /outputs. This is the folder where we will to save the model as it will automatically gets uploaded to IPFS as outputs

` python read_csv.py`: python script to read pandas script

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`. 


```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 11:46:26 [0m[97;40m 61e542a7 [0m[97;40m Docker amancevice/pa... [0m[97;40m Completed [0m[97;40m          [0m[97;40m ipfs://QmY2MEETWyX77... [0m


When it says `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe ${JOB_ID}
```

    Job:
      APIVersion: V1beta1
      Metadata:
        ClientID: 07bde6e8241b19d58c1c5ff3e8ec17e1e80ac6424cd029bd1317a60f1705b583
        CreatedAt: "2023-05-03T11:46:26.767484787Z"
        ID: 61e542a7-bea1-4382-b3c9-40050d143ad6
        Requester:
          RequesterNodeID: QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
          RequesterPublicKey: CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDVRKPgCfY2fgfrkHkFjeWcqno+MDpmp8DgVaY672BqJl/dZFNU9lBg2P8Znh8OTtHPPBUBk566vU3KchjW7m3uK4OudXrYEfSfEPnCGmL6GuLiZjLf+eXGEez7qPaoYqo06gD8ROdD8VVse27E96LlrpD1xKshHhqQTxKoq1y6Rx4DpbkSt966BumovWJ70w+Nt9ZkPPydRCxVnyWS1khECFQxp5Ep3NbbKtxHNX5HeULzXN5q0EQO39UN6iBhiI34eZkH7PoAm3Vk5xns//FjTAvQw6wZUu8LwvZTaihs+upx2zZysq6CEBKoeNZqed9+Tf+qHow0P5pxmiu+or+DAgMBAAE=
      Spec:
        Deal:
          Concurrency: 1
        Docker:
          Entrypoint:
          - python
          - read_csv.py
          Image: amancevice/pandas
          WorkingDirectory: /files
        Engine: Docker
        Language:
          JobContext: {}
        Network:
          Type: None
        Publisher: Estuary
        PublisherSpec:
          Type: Estuary
        Resources:
          GPU: ""
        Timeout: 1800
        Verifier: Noop
        Wasm:
          EntryModule: {}
        inputs:
        - CID: QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz
          Name: ipfs://QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz
          StorageSource: IPFS
          path: /files
        outputs:
        - Name: outputs
          StorageSource: IPFS
          path: /outputs
    State:
      CreateTime: "2023-05-03T11:46:26.767504591Z"
      Executions:
      - AcceptedAskForBid: true
        ComputeReference: e-37a9de63-8bf2-4d83-932f-29fdf98c5274
        CreateTime: "2023-05-03T11:46:35.551431968Z"
        JobID: 61e542a7-bea1-4382-b3c9-40050d143ad6
        NodeId: QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
        PublishedResults:
          CID: QmY2MEETWyX77BBYBNBpUW5bjkVAyP87EotPDVW2vjHG8K
          Name: ipfs://QmY2MEETWyX77BBYBNBpUW5bjkVAyP87EotPDVW2vjHG8K
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
        UpdateTime: "2023-05-03T11:46:35.543450963Z"
        VerificationResult:
          Complete: true
          Result: true
        Version: 6
      - AcceptedAskForBid: true
        ComputeReference: e-1f8a0747-bf6d-49c2-973c-35dfb957448b
        CreateTime: "2023-05-03T11:46:27.267720744Z"
        JobID: 61e542a7-bea1-4382-b3c9-40050d143ad6
        NodeId: QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF
        PublishedResults: {}
        State: BidRejected
        UpdateTime: "2023-05-03T11:46:27.267494495Z"
        VerificationResult: {}
        Version: 3
      - AcceptedAskForBid: true
        ComputeReference: e-8c1c582c-9994-4f87-ba57-746965532790
        CreateTime: "2023-05-03T11:46:28.584477348Z"
        JobID: 61e542a7-bea1-4382-b3c9-40050d143ad6
        NodeId: QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT
        PublishedResults: {}
        State: BidRejected
        UpdateTime: "2023-05-03T11:46:28.5840629Z"
        VerificationResult: {}
        Version: 3
      JobID: 61e542a7-bea1-4382-b3c9-40050d143ad6
      State: Completed
      TimeoutAt: "0001-01-01T00:00:00Z"
      UpdateTime: "2023-05-03T11:46:35.551464593Z"
      Version: 5


When it says `Published` or `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get ${JOB_ID}  --output-dir results
```

    Fetching results of job '61e542a7-bea1-4382-b3c9-40050d143ad6'...
    
    Computing default go-libp2p Resource Manager limits based on:
        - 'Swarm.ResourceMgr.MaxMemory': "34 GB"
        - 'Swarm.ResourceMgr.MaxFileDescriptors': 524288
    
    Applying any user-supplied overrides on top.
    Run 'ipfs swarm limit all' to see the resulting limits.
    
    Results for job '61e542a7-bea1-4382-b3c9-40050d143ad6' have been written to...
    results


## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**,**per_shard files**, and the **raw** directory. To view the file, run the following command:


```bash
%%bash
cat results/stdout # displays the contents of the file
```

                                                    hash  ...  transaction_type
    0  0x04cbcb236043d8fb7839e07bbc7f5eed692fb2ca55d8...  ...                 0
    1  0xcea6f89720cc1d2f46cc7a935463ae0b99dd5fad9c91...  ...                 0
    2  0x463d53f0ad57677a3b430a007c1c31d15d62c37fab5e...  ...                 0
    3  0x05287a561f218418892ab053adfb3d919860988b1945...  ...                 0
    
    [4 rows x 15 columns]

