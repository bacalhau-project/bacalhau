---
sidebar_label: "Data Ingestion"
sidebar_position: 10
---
# How to Ingest Data For Use in Bacalhau

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-ingestion/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-ingestion/index.ipynb)

Before you can start crunching data, you need to make it addressable and accessible via [IPFS](https://ipfs.io/). This notebook will demonstrate several ways to do that.

### Introduction

The goal of the Bacalhau project is to make it easy to perform distributed, decentralised computation next to where the data resides. So a key step in this process is making your data accessible.

IPFS is a set of protocols that allow data to be discovered and accessed in a decentralised way. Data is identified by its content identifier (CID) and can be accessed by anyone who knows the CID. This notebook will show you two ways of interacting with IPFS to move your data from one place (e.g. your machine) to IPFS.

### Prerequisites

* The [Bacalhau CLI](https://docs.bacalhau.org/getting-started/installation) (if you want to run the Bacalhau examples)
* [Docker](https://docs.docker.com/engine/install/) (if you want to run the Docker examples)

## Moving Data via Bacalhau

The easiest way to move data into IPFS is by leveraging helper functions in the Bacalhau CLI.

### URL -> IPFS

The Bacalhau binary includes a helper function to upload from a public URL. This is useful if you have data hosted on a website or in a public S3 bucket (for example).

The following code copies the data from a specified URL to the `/ouputs` directory of a Bacalhau job, and then uploads it to IPFS. Bacalhau will return the CID of the uploaded data.

:::tip

Be careful with the syntax of the command in this example. The `--input-urls` flag only supports writing from a single URL that represents a file to a single path that includes a file. You cannot write to a directory.

To make sure, you can add an `ls` to the command to see what is exposed in the input directory and then download the result and look at the stdout.
:::


```bash
bacalhau docker run --wait --input-urls http://yann.lecun.com/exdb/mnist/train-images-idx3-ubyte.gz:/inputs/train-images-idx3-ubyte.gz ubuntu -- cp -rv /inputs/. /outputs/
```


```python
%env JOB_ID={job_id}
```

    env: JOB_ID=Job ID: 5297c04b-3dea-4f3c-8d06-fc818f919f93
    
    To get the status of the job, run:
      bacalhau describe 5297c04b-3dea-4f3c-8d06-fc818f919f93



```bash
bacalhau list --id-filter ${JOB_ID} --wide
```

The output of the list command presents the CID of the output directory. You can use this in subsequent jobs. For example, let's run a simple command to `ls` the contents of that CID.

:::warning

This file is not pinned. There is no guarantee that the file will exist in the future. If you want to ensure that the file is pinned, use a pinning service.

:::


```bash
bacalhau docker run --download --inputs Qma5e6EDpPe2TsKuz3tumSPSta6vtx48A18f9k99HJATfp ubuntu -- ls -l /inputs/outputs/
```


```bash
cat stdout
```

## Using a Third-Party to Pin Data

If you have data that you want to make available to your Bacalhau jobs (or other people), you can pin it using a pinning service. Pinning services store data on behalf of users and expose the data over IPFS. The pinning provider is essentially guaranteeing that your data will be available if someone knows the CID. Implementation details differ, but the pinning services often use a combination of IPFS nodes and third-party storage providers which are paid for via cryptocurrencies like Filecoin. Most pinning services offer you a free tier, so you can try them out without spending any money.

### Web3.Storage

This example will demonstrate how to pin data using Web3.Storage. Web3.Storage is a pinning service that is built on top of IPFS and Filecoin. It is free to use for small amounts of data, and has a generous free tier. You can find more information about Web3.Storage [here](https://web3.storage/).

#### 1. Create an Account

First you need to create an account (if you don't have one already). Browse to https://web3.storage/login/ and sign up.

#### 2. Sign In and Create an API Key

Next, sign in and browse to the ["Create API Key" page](https://web3.storage/tokens/?create=true). Follow the instructions to create an API key. Once created, you will need to copy the API key to your clipboard.

#### 3. Pin a Local File Using Their Test Client

To test that your API key is working, use [web3.storage's test client to test that it's working](https://bafybeic5r5yxjh5xpmeczfp34ysrjcoa66pllnjgffahopzrl5yhex7d7i.ipfs.dweb.link/).

You can now see (or upload) your file via the web3.storage account page: https://web3.storage/account/.

:::warning
Note that you shouldn't share your API key with anyone. Delete this API key once you have finished with this example.
:::

#### 4. Pin a Local File Via Curl

You can also pin a file via curl. Please view the [API documentation](https://web3.storage/docs/reference/http-api/) to see all available commands. This example submits a single file to be pinned.

```bash
export TOKEN=YOUR_API_KEY
echo hello world > foo.txt
curl -X POST https://api.web3.storage/upload -H "Authorization: Bearer ${TOKEN}" -H "X-NAME: foo.txt" -d @foo.txt
```

#### 5. Pin Multiple Local Files Via Node.JS

Web3.Storage has a [node.js library](https://web3.storage/docs/reference/js-client-library/) to interact with their API. The following example requires node.js to be installed. The following code uses a docker container. The javascript code is located on [their website](https://web3.storage/docs/intro/#create-the-upload-script) or on [github](https://github.com/bacalhau-project/examples/blob/main/data-ingestion/nodejs/put-files.js).

First create some files to upload.


```python
%%writefile nodejs/test1.txt
First test file
```


```python
%%writefile nodejs/test2.txt
Second test file
```

Then run the following command, which uses the environmental variable `TOKEN` to authenticate with the API.

```bash
export TOKEN=YOUR_API_KEY
docker run --rm --env TOKEN=$TOKEN -v $PWD/nodejs:/nodejs node:18-alpine ash -c 'cd /nodejs && npm install && node put-files.js --token=$TOKEN test1.txt test2.txt'
```

```

up to date, audited 245 packages in 706ms

54 packages are looking for funding
  run `npm fund` for details

found 0 vulnerabilities
Uploading 2 files
Content added with CID: bafybeic5smk3bgbsisp566kapp5clmo2ofgmvf223behdpcvjpndpnafka
```

The CID listed at the bottom can now be used as an input to Bacalhau.

#### 6. Pin Files Via the IPFS CLI

See the web3.storage documentation for [instructions on how to pin files via the IPFS CLI](https://web3.storage/docs/how-tos/pinning-services-api/#using-the-ipfs-cli).

#### 7. Pin A File from a URL Via Curl

You can use curl to download a file then re-upload to web3.storage. For example:

```bash
export TOKEN=YOUR_API_KEY
curl -o train-images-idx3-ubyte.gz http://yann.lecun.com/exdb/mnist/train-images-idx3-ubyte.gz
curl -X POST https://api.web3.storage/upload -H "Authorization: Bearer ${TOKEN}" -H "X-NAME: train-images-idx3-ubyte.gz" -d @train-images-idx3-ubyte.gz
```

Which results in something like:

```
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100 9680k  100 9680k    0     0  6281k      0  0:00:01  0:00:01 --:--:-- 6318k
{"cid":"bafybeiereqxn546lkskldoybaa4xe7wk5fricm33nor4oofrxphlaywwd4","carCid":"bagbaieran5ufs752r5vdforovbnjc2gur7kzrsanr3avphsyp7hd6fms7pia"}%  
```

#### 8. Pin A File from a URL Via Node.JS

You can combine the node.js example above with a `wget` to then upload it to web3.storage.

```bash
docker run --rm --env TOKEN=$TOKEN -v $PWD/nodejs:/nodejs node:18-alpine ash -c 'cd /nodejs && wget http://yann.lecun.com/exdb/mnist/train-images-idx3-ubyte.gz && npm install && node put-files.js --token=$TOKEN train-images-idx3-ubyte.gz'
```
