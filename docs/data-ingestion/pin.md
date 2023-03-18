---
sidebar_label: "Pinning data"
sidebar_position: 2
description: "How to pin data to public storage"
---
# Pinning Data

If you have data that you want to make available to your Bacalhau jobs (or other people), you can pin it using a pinning service like Web3.Storage, Estuary, etc. Pinning services store data on behalf of users. The pinning provider is essentially guaranteeing that your data will be available if someone knows the CID. Most pinning services offer you a free tier, so you can try them out without spending any money. 

## Web3.Storage

This example will demonstrate how to pin data using Web3.Storage. Web3.Storage is a pinning service that is built on top of IPFS and Filecoin. It is free to use for small amounts of data, and has a generous free tier. You can find more information about Web3.Storage [here](https://web3.storage/).

- First you need to create an [account](https://web3.storage/login/) (if you don't have one already).
- Next, sign in and browse to the [Create API Key](https://web3.storage/tokens/?create=true) page. Follow the instructions to create an API key. Once created, you will need to copy the API key to your clipboard.

### Ways to pin using web3.storage

1. **Pin a local file using their test client**: To test that your API key is working, use [web3.storage's test client](https://bafybeic5r5yxjh5xpmeczfp34ysrjcoa66pllnjgffahopzrl5yhex7d7i.ipfs.dweb.link/).

You can now see (or upload) your file via the web3.storage [account page](https://web3.storage/account/).

:::warning
Note that you shouldn't share your API key with anyone. Delete this API key once you have finished with this example.
:::

2. **Pin a local file via curl**: You can also pin a file via curl. Please view the [API documentation](https://web3.storage/docs/reference/http-api/) to see all available commands. This example submits a single file to be pinned.

```bash
export TOKEN=YOUR_API_KEY
echo hello world > foo.txt
curl -X POST https://api.web3.storage/upload -H "Authorization: Bearer ${TOKEN}" -H "X-NAME: foo.txt" -d @foo.txt
```

3. **Pin multiple local files via Node.JS**: Web3.Storage has a [node.js library](https://web3.storage/docs/reference/js-client-library/) to interact with their API. The following example requires node.js to be installed. The following code uses a docker container. The javascript code is located on [their website](https://web3.storage/docs/intro/#create-the-upload-script) or on [github](https://github.com/bacalhau-project/examples/blob/main/data-ingestion/nodejs/put-files.js).

First create some files to upload.

```python
%%writefile nodejs/test1.txt
First test file
```

Then run the following command, which uses the environmental variable `TOKEN` to authenticate with the API.

```bash
export TOKEN=YOUR_API_KEY
docker run --rm --env TOKEN=$TOKEN -v $PWD/nodejs:/nodejs node:18-alpine ash -c 'cd /nodejs && npm install && node put-files.js --token=$TOKEN test1.txt test2.txt'
```

The response will return the CID of the file, which can now be used as an input to Bacalhau.

4. **Pin a file from a URL via Curl**: You can use curl to download a file then re-upload to web3.storage. For example:

```bash
export TOKEN=YOUR_API_KEY
curl -o train-images-idx3-ubyte.gz http://yann.lecun.com/exdb/mnist/train-images-idx3-ubyte.gz
curl -X POST https://api.web3.storage/upload -H "Authorization: Bearer ${TOKEN}" -H "X-NAME: train-images-idx3-ubyte.gz" -d @train-images-idx3-ubyte.gz
```


5. **Pin a file from a URL via Node.JS**: You can combine the node.js example above with a `wget` to then upload it to web3.storage.

```bash
docker run --rm --env TOKEN=$TOKEN -v $PWD/nodejs:/nodejs node:18-alpine ash -c 'cd /nodejs && wget http://yann.lecun.com/exdb/mnist/train-images-idx3-ubyte.gz && npm install && node put-files.js --token=$TOKEN train-images-idx3-ubyte.gz'
```

## Estuary

This example show you how to pin data using [estuary](https://estuary.tech/api-admin).

- Before you can upload files via estuary,create an [account](https://estuary.tech) (if you don't have one already).

- Browse to [the API Key management page](https://estuary.tech/api-admin) and create a key.

### Ways to pin using Esturay 

1. **Pin a local file via the Esturay UI**: You can [browse to the Estuary UI](https://estuary.tech/upload) to upload a file via your web browser.

:::tip
Due to the way Estuary batches files for pinning, it may take some time before your file is accessible/listable.
:::

2. **Pin a local file via Curl**: Please view the [API documentation](https://docs.estuary.tech/tutorial-uploading-your-first-file) to see all available commands. This example submits a single file to be pinned.

```bash
export TOKEN=YOUR_API_KEY
echo hello world > foo.txt
curl -X POST https://upload.estuary.tech/content/add -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: multipart/form-data" -F "data=@foo.txt"
```

The response will return the CID of the file.

## View pinned files 

If the upload was successful, you can view the file via your [estuary account page](https://estuary.tech/home). Alternatively, you can obtain this information from the CLI:

```bash
curl -X GET -H "Authorization: Bearer ${TOKEN}" https://api.estuary.tech/content/list
```
