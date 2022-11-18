---
sidebar_label: "Python - Pandas"
sidebar_position: 2
---
# Running Pandas on Bacalhau


[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/python-pandas/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/python-pandas/index.ipynb)

## Introduction

Pandas is a Python package that provides fast, flexible, and expressive data structures designed to make working with "relational" or "labeled" data both easy and intuitive. It aims to be the fundamental high-level building block for doing practical, real world data analysis in Python. Additionally, it has the broader goal of becoming the most powerful and flexible open source data analysis/manipulation tool available in any language. It is already well on its way towards this goal.

### Installing and Getting Started with Pandas



```bash
pip install pandas
```

### Installing Bacalhau

Make sure you have the latest `bacalhau` client installed by following the [getting started instructions](../../../getting-started/installation) or using the hidden installation command below (which installs Bacalhau local to the notebook).


```python
!command -v bacalhau >/dev/null 2>&1 || (export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
path=!echo $PATH
%env PATH=./:{path[0]}
```

    env: PATH=./:./:./:/Users/phil/.pyenv/versions/3.9.7/bin:/opt/homebrew/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/bin:/Users/phil/.gvm/bin:/opt/homebrew/opt/findutils/libexec/gnubin:/opt/homebrew/opt/coreutils/libexec/gnubin:/opt/homebrew/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/bin:/Users/phil/.pyenv/shims:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/Library/TeX/texbin:/usr/local/MacGPG2/bin:/Users/phil/.nexustools


### Installing IPFS

If you are going to upload your data using the IPFS CLI tool then you will need to install that. There are other methods, which you can read more about in the [ingestion example](../../data-ingestion/index.md).

## **Running your pandas script Locally**

#### **Importing data from CSV to DataFrame**
We can also create a DataFrame by importing a CSV file. A CSV file is a text file with one record of data per line. The values within the record are separated using the “comma” character. Pandas provides a useful method, named `read_csv()` to read the contents of the CSV file into a DataFrame. For example, we can create a file named ’`transactions.csv`’ containing details of Transactions. The CSV file is stored in the same directory that contains Python script.



```python
%%writefile read_csv.py
import pandas as pd

print(pd.read_csv("transactions.csv"))
```


```bash
cat read_csv.py
```

    import pandas as pd
    
    print(pd.read_csv("transactions.csv"))


```bash
# Downloading the dataset
wget https://cloudflare-ipfs.com/ipfs/QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz/transactions.csv
```

    --2022-09-15 17:00:57--  https://cloudflare-ipfs.com/ipfs/QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz/transactions.csv
    Resolving cloudflare-ipfs.com (cloudflare-ipfs.com)... 104.17.96.13, 104.17.64.14, 2606:4700::6811:600d, ...
    Connecting to cloudflare-ipfs.com (cloudflare-ipfs.com)|104.17.96.13|:443... connected.
    HTTP request sent, awaiting response... 200 OK
    Length: 1567 (1.5K) [text/csv]
    Saving to: ‘transactions.csv’
    
    transactions.csv    100%[===================>]   1.53K  --.-KB/s    in 0s      
    
    2022-09-15 17:00:58 (37.7 MB/s) - ‘transactions.csv’ saved [1567/1567]
    



```bash
cat transactions.csv
```

    hash,nonce,block_hash,block_number,transaction_index,from_address,to_address,value,gas,gas_price,input,block_timestamp,max_fee_per_gas,max_priority_fee_per_gas,transaction_type
    0x04cbcb236043d8fb7839e07bbc7f5eed692fb2ca55d897f1101eac3e3ad4fab8,12,0x246edb4b351d93c27926f4649bcf6c24366e2a7c7c718dc9158eea20c03bc6ae,483920,0,0x1b63142628311395ceafeea5667e7c9026c862ca,0xf4eced2f682ce333f96f2d8966c613ded8fc95dd,0,150853,50000000000,0xa9059cbb000000000000000000000000ac4df82fe37ea2187bc8c011a23d743b4f39019a00000000000000000000000000000000000000000000000000000000000186a0,1446561880,,,0
    0xcea6f89720cc1d2f46cc7a935463ae0b99dd5fad9c91bb7357de5421511cee49,84,0x246edb4b351d93c27926f4649bcf6c24366e2a7c7c718dc9158eea20c03bc6ae,483920,1,0x9b22a80d5c7b3374a05b446081f97d0a34079e7f,0xf4eced2f682ce333f96f2d8966c613ded8fc95dd,0,150853,50000000000,0xa9059cbb00000000000000000000000066f183060253cfbe45beff1e6e7ebbe318c81e560000000000000000000000000000000000000000000000000000000000030d40,1446561880,,,0
    0x463d53f0ad57677a3b430a007c1c31d15d62c37fab5eee598551697c297c235c,88,0x246edb4b351d93c27926f4649bcf6c24366e2a7c7c718dc9158eea20c03bc6ae,483920,2,0x9df428a91ff0f3635c8f0ce752933b9788926804,0x9e669f970ec0f49bb735f20799a7e7c4a1c274e2,11000440000000000,90000,50000000000,0x,1446561880,,,0
    0x05287a561f218418892ab053adfb3d919860988b19458c570c5c30f51c146f02,20085,0x246edb4b351d93c27926f4649bcf6c24366e2a7c7c718dc9158eea20c03bc6ae,483920,3,0x2a65aca4d5fc5b5c859090a6c34d164135398226,0x743b8aeedc163c0e3a0fe9f3910d146c48e70da8,1530219620000000000,90000,50000000000,0x,1446561880,,,0

### Running the script



```bash
python3 read_csv.py
```

                                                    hash  ...  transaction_type
    0  0x04cbcb236043d8fb7839e07bbc7f5eed692fb2ca55d8...  ...                 0
    1  0xcea6f89720cc1d2f46cc7a935463ae0b99dd5fad9c91...  ...                 0
    2  0x463d53f0ad57677a3b430a007c1c31d15d62c37fab5e...  ...                 0
    3  0x05287a561f218418892ab053adfb3d919860988b1945...  ...                 0
    
    [4 rows x 15 columns]


## **Running the script on bacalhau**

To run pandas on bacalhau you must upload your datasets along with the script to IPFS this can be done by using the IPFS CLI to upload the files or using a pinning service like pinata or nft.storage

Adding the Scripts and Datasets to IPFS
since we already uploaded these scripts to IPFS there is no need for you to add them

```
$ ipfs add -r .
added QmPqx4BaWzAmZm4AuBqGtG6dkX7bGSVgjfgpkv2g7mi3uz pandas/read_csv.py
added QmYErPqtdpNTxpKot9pXR5QbhGSyaGdMFxfUwGHm4rzXzH pandas/transactions.csv
added QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz pandas
 1.59 KiB / 1.59 KiB [===================================================================================]
```


For running pandas in bacalhau you need choose a container which has python and pandas Installed

Structure of the bacalhau command

`bacalhau docker run ` similar to docker run

-v mount the CID to the container this is the 

CID:/&lt;PATH-TO-WHERE-THE-CID-IS-TO-BE-MOUNTED> `QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz:/files`

-w is used to set the working directory

-- /bin/bash -c 'python hello.py' (Run the script)

### Command:


```bash
 bacalhau  docker run \
--wait \
--id-only \
-v QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz:/files \
-w /files \
amancevice/pandas \
-- python read_csv.py
```

    e6377c99-b637-4661-a334-6ce98fcf037c



```python
%env JOB_ID={job_id}
```

Running the commands will output a UUID (like `e6377c99-b637-4661-a334-6ce98fcf037c`). This is the ID of the job that was created. You can check the status of the job with the following command:




```bash
bacalhau list --id-filter ${JOB_ID}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 17:07:18 [0m[97;40m e6377c99 [0m[97;40m Docker amancevice/pa... [0m[97;40m Published [0m[97;40m          [0m[97;40m /ipfs/bafybeihaqoxj7... [0m



Where it says "`Published`", that means the job is done, and we can get the results.

If there is an error you can view the error using the following command bacalhau describe


```bash
bacalhau describe ${JOB_ID}
```

Since there is no error we can’t see any error instead we see the state of our job to be complete

we create a temporary directory to save our results


```bash
mkdir pandas-results
```

To Download the results of your job, run the following command:


```bash
bacalhau get ${JOB_ID}  --output-dir pandas-results
```

    [90m17:14:05.466 |[0m [32mINF[0m [1mbacalhau/get.go:67[0m[36m >[0m Fetching results of job 'e6377c99-b637-4661-a334-6ce98fcf037c'...
    2022/09/15 17:14:06 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.
    [90m17:14:16.401 |[0m [32mINF[0m [1mipfs/downloader.go:115[0m[36m >[0m Found 1 result shards, downloading to temporary folder.
    [90m17:14:21.283 |[0m [32mINF[0m [1mipfs/downloader.go:195[0m[36m >[0m Combining shard from output volume 'outputs' to final location: '/content/pandas-results'


After the download has finished you should 
see the following contents in pandas-results directory


```bash
ls pandas-results/combined_results/
```

    shards	stderr	stdout	volumes


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
cat pandas-results/combined_results/stdout
```

                                                    hash  ...  transaction_type
    0  0x04cbcb236043d8fb7839e07bbc7f5eed692fb2ca55d8...  ...                 0
    1  0xcea6f89720cc1d2f46cc7a935463ae0b99dd5fad9c91...  ...                 0
    2  0x463d53f0ad57677a3b430a007c1c31d15d62c37fab5e...  ...                 0
    3  0x05287a561f218418892ab053adfb3d919860988b1945...  ...                 0
    
    [4 rows x 15 columns]

