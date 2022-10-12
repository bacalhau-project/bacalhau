---
sidebar_label: BlockchainETL
sidebar_position: 3
---
[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-engineering/blockchain-etl/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-engineering/blockchain-etl/index.ipynb)

# BlockchainETL


# **Introduction**

Analyzing and loading blockchain data can be difficult since loading blockchain ledgers can be hard to do since downloading them for performing ETL can be difficult as well tricky, since the datasets are on IPFS we could mount the CIDs and then analyze the performing ETL operations on that data can be made easier to do it at scale,[ Ethereum ETL](https://ethereum-etl.readthedocs.io/en/latest/) lets you convert blockchain data into convenient formats like CSVs and relational databases.

Prerequisites



* Python 3 running locally - [Installation Binaries](http://here) / [Tutorial](https://realpython.com/installing-python/)
* The Bacalhau client - [Installation instructions](https://docs.bacalhau.org/getting-started/installation)
* (Optional, but Recommended) Docker - [Installation Instructions](https://docs.docker.com/get-docker/)


# **Running EthereumETL locally**

Downloading Datasets

Data could be downloaded using tools like [Geth Documentation | Go Ethereum](https://geth.ethereum.org/docs/) and then you can upload it to IPFS or better use the dataset already there on IPFS link to the dataset used   

For this example you can use this CSV file



```bash
wget https://cloudflare-ipfs.com/ipfs/QmTAQMGiSv9xocaB4PUCT5nSBHrf9HZrYj21BAZ5nMTY2W/transactions.csv
```

    --2022-10-03 04:44:33--  https://cloudflare-ipfs.com/ipfs/QmTAQMGiSv9xocaB4PUCT5nSBHrf9HZrYj21BAZ5nMTY2W/transactions.csv
    Resolving cloudflare-ipfs.com (cloudflare-ipfs.com)... 104.17.96.13, 104.17.64.14, 2606:4700::6811:400e, ...
    Connecting to cloudflare-ipfs.com (cloudflare-ipfs.com)|104.17.96.13|:443... connected.
    HTTP request sent, awaiting response... 200 OK
    Length: 1567 (1.5K) [text/csv]
    Saving to: ‘transactions.csv’
    
    transactions.csv    100%[===================>]   1.53K  --.-KB/s    in 0s      
    
    2022-10-03 04:44:34 (21.5 MB/s) - ‘transactions.csv’ saved [1567/1567]
    



Create a folder called ‘outputs’ for the output dataset



```bash
 mkdir outputs
```

Installing the ethreumetl package


```bash
 pip install ethereum-etl
```


Run the following command  for Extracting transaction hashes from `transactions.csv`




```bash
 ethereumetl extract_csv_column --input transactions.csv --column hash --output ./output/transaction_hashes.csv
```

    [0m


Source [Commands - Ethereum ETL](https://ethereum-etl.readthedocs.io/en/latest/commands/) For running other commands



```bash
 cat output/transaction_hashes.csv
```

    0x04cbcb236043d8fb7839e07bbc7f5eed692fb2ca55d897f1101eac3e3ad4fab8
    0xcea6f89720cc1d2f46cc7a935463ae0b99dd5fad9c91bb7357de5421511cee49
    0x463d53f0ad57677a3b430a007c1c31d15d62c37fab5eee598551697c297c235c
    0x05287a561f218418892ab053adfb3d919860988b19458c570c5c30f51c146f02





## **Creating a docker container**

In this step you will create a  `Dockerfile` to create your Docker deployment. The `Dockerfile` is a text document that contains the commands used to assemble the image.

First, create the `Dockerfile`.

Next, add your desired configuration to the `Dockerfile`. These commands specify how the image will be built, and what extra requirements will be included.


```
FROM python:3.8

RUN pip install ethereum-etl
```


We create a simple python container with just installing the single package [Ethereum ETL](https://ethereum-etl.readthedocs.io/en/latest/)

Build the container


```
docker build -t <hub-user>/<repo-name>:<tag> .
```


Please replace

&lt;hub-user> with your docker hub username, If you don’t have a docker hub account [Follow these instructions to create docker account](https://docs.docker.com/docker-id/), and use the username of the account you created

&lt;repo-name> This is the name of the container, you can name it anything you want

&lt;tag> This is not required but you can use the latest tag

After you have build the container, the next step is to test it locally and then push it docker hub

Before pushing you first need to create a repo which you can create by following the instructions here [https://docs.docker.com/docker-hub/repos/](https://docs.docker.com/docker-hub/repos/)

Now you can push this repository to the registry designated by its name or tag.


```
 docker push <hub-user>/<repo-name>:<tag>
```


After the repo image has been pushed to docker hub, we can now use the container for running on bacalhau


# **Running EthereumETL on bacalhau**

Structure of the Command

`bacalhau docker run ` similar to docker run

-v mount the CID to the container this is the 

CID:/&lt;PATH-TO-WHERE-THE-CID-IS-TO-BE-MOUNTED> `QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz:/files`

-- **`ethereumetl extract_csv_column --input transactions.csv --column hash --output ./output/transaction_hashes.csv`** running the command on bacalhau

Command:


```
bacalhau docker run \
-v QmYErPqtdpNTxpKot9pXR5QbhGSyaGdMFxfUwGHm4rzXzH:/transactions.csv \
jsace/ethereum-etl \
-- ethereumetl extract_csv_column --input transactions.csv --column hash --output ./output/transaction_hashes.csv
```



Insalling bacalhau


```bash
 curl -sL https://get.bacalhau.org/install.sh | bash
```

    Your system is linux_amd64
    No BACALHAU detected. Installing fresh BACALHAU CLI...
    Getting the latest BACALHAU CLI...
    Installing v0.2.5 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.2.5/bacalhau_v0.2.5_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.2.5/bacalhau_v0.2.5_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into /usr/local/bin successfully.
    Client Version: v0.2.5
    Server Version: v0.2.5



```bash
echo $(bacalhau docker run --id-only --wait --wait-timeout-secs 1000 -v QmYErPqtdpNTxpKot9pXR5QbhGSyaGdMFxfUwGHm4rzXzH:/transactions.csv jsace/ethereum-etl -- ethereumetl extract_csv_column --input transactions.csv --column hash --output ./outputs/transaction_hashes.csv) > job_id.txt
cat job_id.txt
```

    75ef84c5-1f39-483f-a33f-508c9f7a789a



Running the commands will output a UUID (like `54506541-4eb9-45f4-a0b1-ea0aecd34b3e`). This is the ID of the job that was created. You can check the status of the job with the following command:



```bash
bacalhau list --id-filter $(cat job_id.txt)
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 04:45:02 [0m[97;40m 75ef84c5 [0m[97;40m Docker jsace/ethereu... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmRcanuDamGtJz... [0m



Where it says "`Published `", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```bash
bacalhau describe $(cat job_id.txt)
```

    JobAPIVersion: ""
    ID: 75ef84c5-1f39-483f-a33f-508c9f7a789a
    RequesterNodeID: QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
    ClientID: 14748225207ab2b2535e2f04ecd0ed2b1ac44363d6ef419ba05cca30377d6aca
    Spec:
        Engine: 2
        Verifier: 1
        Publisher: 4
        Docker:
            Image: jsace/ethereum-etl
            Entrypoint:
                - ethereumetl
                - extract_csv_column
                - --input
                - transactions.csv
                - --column
                - hash
                - --output
                - ./outputs/transaction_hashes.csv
        inputs:
            - Engine: 1
              Cid: QmYErPqtdpNTxpKot9pXR5QbhGSyaGdMFxfUwGHm4rzXzH
              path: /transactions.csv
        outputs:
            - Engine: 1
              Name: outputs
              path: /outputs
        Sharding:
            BatchSize: 1
            GlobPatternBasePath: /inputs
    Deal:
        Concurrency: 1
    CreatedAt: 2022-10-03T04:45:02.775669316Z
    JobState:
        Nodes:
            QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3:
                Shards:
                    0:
                        NodeId: QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
                        ShardIndex: 0
                        State: 7
                        Status: 'Got results proposal of length: 0'
                        VerificationProposal: []
                        VerificationResult:
                            Complete: true
                            Result: true
                        PublishedResults:
                            Engine: 1
                            Name: job-75ef84c5-1f39-483f-a33f-508c9f7a789a-shard-0-host-QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
                            Cid: QmRcanuDamGtJzYreoP79ButPimGYo6oz2v1T7FPr6GRhP
                        RunOutput:
                            Stdout: ""
                            StdoutTruncated: false
                            Stderr: ""
                            StderrTruncated: false
                            ExitCode: 0
                            RunnerError: ""


Since there is no error we can’t see any error instead we see the state of our job to be complete, that means 
we can download the results!
we create a temporary directory to save our results


```bash
mkdir results
```

To Download the results of your job, run 

---

the following command:


```bash
bacalhau get  $(cat job_id.txt)  --output-dir results
```

    [90m04:45:07.642 |[0m [32mINF[0m [1mbacalhau/get.go:67[0m[36m >[0m Fetching results of job '75ef84c5-1f39-483f-a33f-508c9f7a789a'...
    2022/10/03 04:45:08 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.
    [90m04:45:18.137 |[0m [32mINF[0m [1mipfs/downloader.go:115[0m[36m >[0m Found 1 result shards, downloading to temporary folder.
    [90m04:46:19 |[0m [32mINF[0m [1mipfs/downloader.go:195[0m[36m >[0m Combining shard from output volume 'outputs' to final location: '/content/results'


After the download has finished you should 
see the following contents in results directory


```bash
ls results/
```

    shards	stderr	stdout	volumes


VIEWING THE RESULTS CSV


```bash
cat results/volumes/outputs/transaction_hashes.csv
```

    0x04cbcb236043d8fb7839e07bbc7f5eed692fb2ca55d897f1101eac3e3ad4fab8
    0xcea6f89720cc1d2f46cc7a935463ae0b99dd5fad9c91bb7357de5421511cee49
    0x463d53f0ad57677a3b430a007c1c31d15d62c37fab5eee598551697c297c235c
    0x05287a561f218418892ab053adfb3d919860988b19458c570c5c30f51c146f02



```bash
bacalhau describe $(cat job_id.txt) --spec > job.yaml
```


```bash
cat job.yaml
```

    JobAPIVersion: ""
    ID: 75ef84c5-1f39-483f-a33f-508c9f7a789a
    RequesterNodeID: QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
    ClientID: 14748225207ab2b2535e2f04ecd0ed2b1ac44363d6ef419ba05cca30377d6aca
    Spec:
        Engine: 2
        Verifier: 1
        Publisher: 4
        Docker:
            Image: jsace/ethereum-etl
            Entrypoint:
                - ethereumetl
                - extract_csv_column
                - --input
                - transactions.csv
                - --column
                - hash
                - --output
                - ./outputs/transaction_hashes.csv
        inputs:
            - Engine: 1
              Cid: QmYErPqtdpNTxpKot9pXR5QbhGSyaGdMFxfUwGHm4rzXzH
              path: /transactions.csv
        outputs:
            - Engine: 1
              Name: outputs
              path: /outputs
        Sharding:
            BatchSize: 1
            GlobPatternBasePath: /inputs
    Deal:
        Concurrency: 1
    CreatedAt: 2022-10-03T04:45:02.782847165Z
    JobState:
        Nodes:
            QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3:
                Shards:
                    0:
                        NodeId: QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
                        ShardIndex: 0
                        State: 7
                        Status: 'Got results proposal of length: 0'
                        VerificationProposal: []
                        VerificationResult:
                            Complete: true
                            Result: true
                        PublishedResults:
                            Engine: 1
                            Name: job-75ef84c5-1f39-483f-a33f-508c9f7a789a-shard-0-host-QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
                            Cid: QmRcanuDamGtJzYreoP79ButPimGYo6oz2v1T7FPr6GRhP
                        RunOutput:
                            Stdout: ""
                            StdoutTruncated: false
                            Stderr: ""
                            StderrTruncated: false
                            ExitCode: 0
                            RunnerError: ""

