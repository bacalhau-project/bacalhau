---
sidebar_label: 'Onboard Your Workload'
sidebar_position: 2
---
import ReactPlayer from 'react-player'

# Onboarding Your Workloads

This page describes how to convert your workload into a Bacalhau format. To migrate your workload, follow the instructions below for the job format you want to use. Also, don't forget to check out the examples for more inspiration.

:::tip

We will be adding more job formats soon!

:::

## Docker

These instructions show you how to migrate a workload that is based on a Docker container into a format that will work with Bacalhau.

### Prerequisites and Limitations

To help provide a safe, secure network for all users we add several runtime restrictions:

1. All ingress/egress networking is disabled. You won't be able to pull data/code/weights/etc. from an external source.
2. Data passing is implemented using Docker volumes, using [Bacalhau's input/output volumes](../about-bacalhau/architecture.md#input--output-volumes).

The following lists current limitations:

* Public container registries only
* Containers must have an `x86_64` CPU architecture
* The `--inputs` and `--input-voumes` flags do not support CID subpaths. Directories only.
* The `--input-urls` flag does not support URL directories. Single files only.

### Onboarding

#### 1. (Optional) Read Data From the `/inputs` Directory

If you need to pass data into your container, you will do this via a Docker volume, so you need to modify your code to read from a local directory.

We assume you are reading from a directory called `/inputs`, which is the default.

:::tip

You can specify which directory the data is written to with the `--input-volumes` CLI flag.

:::

#### 2. (Optional) Write Data to the `/outputs` Directory

If you need to return data from your container, you will do this via a Docker volume, so you need to modify your code to write to a local directory.

We assume you are writing to a directory called `/outputs`, which is the default.

:::tip

You can specify which directory the data is written to with the `--output-volumes` CLI flag.

:::

#### 3. (Optional) Write Data To an Output Directory

If you need to pass data into your container, you will do this via a Docker volume, so you need to modify your code to read from a local directory.

#### 4. (Optional) Build and Push Your Image To a Public Registry

If you haven't already, [build your image](https://docs.docker.com/engine/reference/commandline/build/) and [push it](https://docs.docker.com/engine/reference/commandline/push/) to a publicly accessible container registry.

:::caution

All Bacalhau nodes are of an `x86_64` architecture, therefore containers must be built for `x86_64` systems.

:::

For example:

```bash
export IMAGE=myuser/myimage:latest
docker build -t ${IMAGE} .
docker image push ${IMAGE}
```

#### 5. Test Your Container

Execute the following command to test your docker image locally, changing the environment variables as necessary:

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

Results in:

```bash
...file listing...
do something useful
```

#### 6. (Optional) Upload the Input Data to IPFS

We recommend uploading your data to IPFS for persistent storage:

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

#### 7. Run the Workload on Bacalhau

To run your workload using input data stored in IPFS use the following command:

```bash
bacalhau docker run --inputs ${CID} ${IMAGE} ${CMD}

bacalhau list 

bacalhau get JOB_ID
```

For example, running:

```bash
cid=$(bacalhau docker run ubuntu echo hello)
bacalhau list --id-filter $cid
sleep 5
bacalhau list --id-filter $cid
bacalhau get $cid
ls shards
```

Results in:

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

The `--inputs` flag does not support CID subpaths. Directories only.

:::

Alternatively, run your workload with a publicly accessible http(s) URL, which will download the data temporarily into IPFS:

```bash
export URL=https://download.geofabrik.de/antarctica-latest.osm.pbf
bacalhau docker run --input-urls ${URL}:/inputs ${IMAGE} ${CMD}

bacalhau list 

bacalhau get JOB_ID
```

:::caution

The `--input-urls` flag does not support URL directories. Single files only.

:::

### Examples

Here is an example of an onboarded workload leveraging the Surface Ocean CO₂ Atlas (SOCAT) to Bacalhau:
- [Youtube: Bacalhau SOCAT Workload Demo](https://www.youtube.com/watch?v=t2AHD8yJhLY)
- [Github: bacalhau_socat_test](https://github.com/wesfloyd/bacalhau_socat_test)

<!-- <ReactPlayer playing controls url='https://www.youtube.com/watch?v=t2AHD8yJhLY' playing='false'/> -->

Here is an example of running a job live on the Bacalhau network: [Youtube: Bacalhau Intro Video](https://www.youtube.com/watch?v=wkOh05J5qgA)

## Support

Please reach out to the [Bacalhau team via Slack](https://filecoinproject.slack.com/archives/C02RLM3JHUY) if you would like help pinning data to IPFS for your job or in case of any issues.
