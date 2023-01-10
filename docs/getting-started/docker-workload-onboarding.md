---
sidebar_label: 'Onboard Docker Workload'
sidebar_position: 2
description: How to use docker containers with Bacalhau
---
import ReactPlayer from 'react-player'

# Onboarding Your Docker Workloads

Bacalhau executes jobs by running them within containers. This sections describes how to migrate a workload based on a Docker container into a format that will work with the Bacalhau client. 

:::tip

You can check out this example tutorial on [how to work with custom containers in Bacalhau](https://docs.bacalhau.org/examples/workload-onboarding/custom-containers/) to see how we used all these steps together. 

:::

## Requirements
Here are some few things to note before getting started:
* You must publish the container to a public container registry that is accessible from the Bacalhau network
* Bacalhau supports only `amd64` images. Does not support `arm64` images
* Containers must have an `x86_64` CPU architecture
* The `--inputs` and `--input-volumes` flags do not support CID subpaths only **directories** 
* The `--input-urls` flag does not support URL directories only **single files** only

:::tip

You can check to see a [list of example public containers](https://github.com/orgs/bacalhau-project/packages?repo_name=examples) used by the Bacalhau team

**Note**: Only about a third of examples have their containers here. The rest are under random docker hub registries (mostly Vedants).

:::

## Runtime Restrictions

To help provide a safe, secure network for all users, we add the following runtime restrictions:

- All ingress/egress networking is disabled. You won't be able to pull `data/code/weights/` etc from an external source
- Data passing is implemented with Docker volumes, using [Bacalhau's input/output volumes](https://docs.bacalhau.org/about-bacalhau/architecture#input--output-volumes)


## Onboarding Your Workload

### Step 1 - Read Data From Your Directory

If you need to pass data into your container you will do this through a Docker volume. You'll need to modify your code to read from a local directory.

We make the assumption that you are reading from a directory called `/inputs`, which is set as the default.

:::tip

You can specify which directory the data is written to with the [`--input-volumes`](https://docs.bacalhau.org/all-flags#run-python) CLI flag.

:::

### Step 2 - Write Data to the Your Directory

If you need to return data from your container you will do this through a Docker volume. You'll need to modify your code to write to a local directory.

We make the assumption that you are writing to a directory called `/outputs`, which is set as the default.

:::tip

You can specify which directory the data is written to with the [`--output-volumes`](https://docs.bacalhau.org/all-flags#run-python) CLI flag.

:::

### Step 3 - Build and Push Your Image To a Registry

If you haven't already, you'll need to [build your image](https://docs.docker.com/engine/reference/commandline/build/) and [push it](https://docs.docker.com/engine/reference/commandline/push/) to a publicly accessible container registry.

:::caution

All Bacalhau nodes are of an `x86_64` architecture, therefore containers must be built for [`x86_64` systems](#what-containers-to-use).

:::

For example:

```bash
export IMAGE=myuser/myimage:latest
docker build -t ${IMAGE} .
docker image push ${IMAGE}
```

### Step 4 - Test Your Container

To test your docker image locally, you'll need to execute the following command, changing the environment variables as necessary:

```bash
export LOCAL_INPUT_DIR=$PWD
export LOCAL_OUTPUT_DIR=$PWD
export CMD=(sh -c 'ls /inputs; echo do something useful > /outputs/stdout')
docker run --rm \
  -v ${LOCAL_INPUT_DIR}:/inputs  \
  -v ${LOCAL_OUTPUT_DIR}:/outputs \
  ${IMAGE} \
  ${CMD}
```

For example:

```bash
export IMAGE=ubuntu
docker run --rm \
  -v ${LOCAL_INPUT_DIR}:/inputs  \
  -v ${LOCAL_OUTPUT_DIR}:/outputs \
  ${IMAGE} \
  ${CMD}
cat stdout
```

This snippet results in:

```bash
...file listing...
do something useful
```

### Step 5 - Upload the Input Data

We recommend uploading your data to IPFS for persistent storage, because:

* Bacalhau is designed to perform the computation next to the data
* Distributing data across the solar system with IPFS distributes the Bacalhau computation
* Distributing computation improves performance by scaling, and improves resiliency via redundancy
* Using IPFS CIDs as inputs enables repeatable and cacheable execution

:::tip

The following guides explain how to store data on the IPFS network.

- Leverage an IPFS “pinning service” such as:
  - [Web3.Storage](https://web3.storage/account/)
  - [Estuary](https://estuary.tech/sign-in)
  - [Manually pin your files to IPFS](https://docs.ipfs.io/how-to/pin-files/) with your own IPFS server.
- If uploading a folder of input files, consider [uploading with this script](https://web3.storage/docs/#create-the-upload-script). However, please note that any content uploaded to Web3.storage is [also wrapped in a parent directory](https://web3.storage/docs/how-tos/store/#directory-wrapping). You will need to take care to reference the inner directory CID in your bacalhau command.

:::

### Step 6 - Run the Workload on Bacalhau

To run your workload using input data stored in IPFS, run the following command:

```bash
bacalhau docker run --inputs ${CID} ${IMAGE} ${CMD}

bacalhau list 

bacalhau get JOB_ID
```

For example, running:

```bash
job_id=$(bacalhau docker run ubuntu echo hello)
bacalhau list --id-filter $job_id
sleep 5
bacalhau list --id-filter $job_id
bacalhau get $job_id
ls shards
```

outputs:

```bash
 CREATED   ID        JOB                      STATE      VERIFIED  PUBLISHED 
 10:26:00  24440f0d  Docker ubuntu echo h...  Verifying                      
 CREATED   ID        JOB                      STATE      VERIFIED  PUBLISHED               
 10:26:00  24440f0d  Docker ubuntu echo h...  Published            /ipfs/bafybeiflj3kha... 
11:26:09.107 | INF bacalhau/get.go:67 > Fetching results of job '24440f0d-3c06-46af-9adf-cb524aa43961'...
11:26:10.528 | INF ipfs/downloader.go:115 > Found 1 result shards, downloading to temporary folder.
11:26:13.144 | INF ipfs/downloader.go:195 > Combining shard from output volume 'outputs' to final location: '/Users/phil/source/filecoin-project/docs.bacalhau.org'
job-24440f0d-3c06-46af-9adf-cb524aa43961-shard-0-host-QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
```

:::caution

The `--inputs` flag does not support CID subpaths.

:::

Alternatively, you can run your workload with a publicly accessible http(s) URL, which will download the data temporarily into IPFS:

```bash
export URL=https://download.geofabrik.de/antarctica-latest.osm.pbf
bacalhau docker run --input-urls ${URL}:/inputs ${IMAGE} ${CMD}

bacalhau list 

bacalhau get JOB_ID
```

:::caution

The `--input-urls` flag does not support URL directories.

:::

## Video Tutorials

We have video tutorial examples on how to onboarded workload to Bacalhau:
- [Youtube: Bacalhau SOCAT Workload Demo](https://www.youtube.com/watch?v=t2AHD8yJhLY)
- [Github: bacalhau_socat_test](https://github.com/wesfloyd/bacalhau_socat_test)
- [Youtube: Bacalhau Intro Video](https://www.youtube.com/watch?v=wkOh05J5qgA)

<!-- <ReactPlayer playing controls url='https://www.youtube.com/watch?v=t2AHD8yJhLY' playing='false'/> -->

## Troubleshooting
If you run into this compute error while running your docker image 

```bash
Creating job for submission ... done ✅
Finding node(s) for the job ... done ✅
Node accepted the job ... done ✅
Error while executing the job.
```

This can often be resolved by re-tagging your docker image

## Support

Please reach out to the [Bacalhau team via Slack](https://filecoinproject.slack.com/archives/C02RLM3JHUY) if you would like help pinning data to IPFS for your job or for any issues you encounter.
