---
sidebar_label: "Pinning to Filecoin"
sidebar_position: 2
description: "How to pin data to IPFS using filecoin"
---
# Pinning Data to IPFS with Filecoin 

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-ingestion/pinning/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-ingestion/pinning/index.ipynb)

Before you can start crunching data, you need to make it addressable and accessible via IPFS. [IPFS](https://ipfs.io/) is a set of protocols that allow data to be discovered and accessed in a decentralised way. Data is identified by its content identifier (CID) and can be accessed by anyone who knows the CID. 

The goal of the Bacalhau project is to make it easy to perform distributed, decentralised computation next to where the data resides. So a key step in this process is making your data accessible. This tutorial shows how to pin data to IPFS using [Filecoin](https://filecoin.io/).


## Using a Third-Party to Pin Data

If you have data that you want to make available to your Bacalhau jobs (or other people), you can pin it using a pinning service like Web3.Storage, Estuary, etc. Pinning services store data on behalf of users and expose the data over IPFS. The pinning provider is essentially guaranteeing that your data will be available if someone knows the CID. Implementation details differ, but the pinning services often use a combination of IPFS nodes and third-party storage providers which are paid for via cryptocurrencies like Filecoin. Most pinning services offer you a free tier, so you can try them out without spending any money. 

For the course of this tutorial, we will explore how to use Web3.Storage and Estuary pinning services to upload data onto Filecoin and pin it to IPFS.


## Web3.Storage

This example will demonstrate how to pin data using Web3.Storage. Web3.Storage is a pinning service that is built on top of IPFS and Filecoin. It is free to use for small amounts of data, and has a generous free tier. You can find more information about Web3.Storage [here](https://web3.storage/).

### 1. Create an Account

First you need to create an account (if you don't have one already). Browse to https://web3.storage/login/ and sign up.

### 2. Sign In and Create an API Key

Next, sign in and browse to the ["Create API Key" page](https://web3.storage/tokens/?create=true). Follow the instructions to create an API key. Once created, you will need to copy the API key to your clipboard.

### 3. Pin a Local File Using Their Test Client

To test that your API key is working, use [web3.storage's test client to test that it's working](https://bafybeic5r5yxjh5xpmeczfp34ysrjcoa66pllnjgffahopzrl5yhex7d7i.ipfs.dweb.link/).

You can now see (or upload) your file via the web3.storage account page: https://web3.storage/account/.

:::warning
Note that you shouldn't share your API key with anyone. Delete this API key once you have finished with this example.
:::

### 4. Pin a Local File Via Curl

You can also pin a file via curl. Please view the [API documentation](https://web3.storage/docs/reference/http-api/) to see all available commands. This example submits a single file to be pinned.

```bash
export TOKEN=YOUR_API_KEY
echo hello world > foo.txt
curl -X POST https://api.web3.storage/upload -H "Authorization: Bearer ${TOKEN}" -H "X-NAME: foo.txt" -d @foo.txt
```

### 5. Pin Multiple Local Files Via Node.JS

Web3.Storage has a [node.js library](https://web3.storage/docs/reference/js-client-library/) to interact with their API. The following example requires node.js to be installed. The following code uses a docker container. The javascript code is located on [their website](https://web3.storage/docs/intro/#create-the-upload-script) or on [github](https://github.com/bacalhau-project/examples/blob/main/data-ingestion/nodejs/put-files.js).

First create some files to upload.


```python
%%writefile nodejs/test1.txt
First test file
```

    Overwriting nodejs/test1.txt



```python
%%writefile nodejs/test2.txt
Second test file
```

    Overwriting nodejs/test2.txt


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

### 6. Pin Files Via the IPFS CLI

See the web3.storage documentation for [instructions on how to pin files via the IPFS CLI](https://web3.storage/docs/how-tos/pinning-services-api/#using-the-ipfs-cli).

### 7. Pin A File from a URL Via Curl

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

### 8. Pin A File from a URL Via Node.JS

You can combine the node.js example above with a `wget` to then upload it to web3.storage.

```bash
docker run --rm --env TOKEN=$TOKEN -v $PWD/nodejs:/nodejs node:18-alpine ash -c 'cd /nodejs && wget http://yann.lecun.com/exdb/mnist/train-images-idx3-ubyte.gz && npm install && node put-files.js --token=$TOKEN train-images-idx3-ubyte.gz'
```

## Estuary

This example show you how to pin data using [https://estuary.tech](https://estuary.tech/api-admin).

### 1. Create an Account

Before you can upload files via estuary, you need an account. [Sign up](https://estuary.tech).

### 2. Create an API Key

Browse to [the API Key mangement page](https://estuary.tech/api-admin) and create a key.

### 3. Pin a Local File via the Esturay UI

You can [browse to the Estuary UI](https://estuary.tech/upload) to upload a file via your web browser.

:::tip

Due to the way Estuary batches files for pinning, it may take some time before your file is accessible/listable.

:::

### 4. Pin a Local File Via Curl

Please view the [API documentation](https://docs.estuary.tech/tutorial-uploading-your-first-file) to see all available commands. This example submits a single file to be pinned.

```bash
export TOKEN=YOUR_API_KEY
echo hello world > foo.txt
curl -X POST https://upload.estuary.tech/content/add -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: multipart/form-data" -F "data=@foo.txt"
```

The response will return the CID of the file.

### 5. View Pinned Files

If the upload was successful, you can view the file via your [estuary account page](https://estuary.tech/home).

Alternatively, you can obtain this information from the CLI:

```bash
curl -X GET -H "Authorization: Bearer ${TOKEN}" https://api.estuary.tech/content/list
```
