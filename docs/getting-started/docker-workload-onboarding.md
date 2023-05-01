---
sidebar_label: 'Onboard Docker Workload'
sidebar_position: 3
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
* The `--input ipfs://...` flag does not support CID subpaths only **directories** 
* The `--input https://...` flag does not support URL directories only **single files** only
* The `--input s3://...` flag does support S3 keys and prefixes. e.g. `s3://bucket/logs-2023-04*` for all April 2023 logs 

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

You can specify which directory the data is written to with the [`--input`](https://docs.bacalhau.org/all-flags#run-python) CLI flag.

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

```shell
$ export IMAGE=myuser/myimage:latest
$ docker build -t ${IMAGE} .
$ docker image push ${IMAGE}
```

### Step 4 - Test Your Container

To test your docker image locally, you'll need to execute the following command, changing the environment variables as necessary:

```shell
$ export LOCAL_INPUT_DIR=$PWD
$ export LOCAL_OUTPUT_DIR=$PWD
$ export CMD=(sh -c 'ls /inputs; echo do something useful > /outputs/stdout')
$ docker run --rm \
  -v ${LOCAL_INPUT_DIR}:/inputs  \
  -v ${LOCAL_OUTPUT_DIR}:/outputs \
  ${IMAGE} \
  ${CMD}
```

For example:

```shell
$ export IMAGE=ubuntu
$ docker run --rm \
  -v ${LOCAL_INPUT_DIR}:/inputs  \
  -v ${LOCAL_OUTPUT_DIR}:/outputs \
  ${IMAGE} \
  ${CMD}
$ cat stdout
```

This snippet results in:

```
...file listing...
do something useful
```

### Step 5 - Upload the Input Data

Data is identified by its content identifier (CID) and can be accessed by anyone who knows the CID. You can use either of these methods to upload your data:

- [Copy data from a URL to public storage](https://docs.bacalhau.org/data-ingestion/from-url)
- [Pin Data to public storage](https://docs.bacalhau.org/data-ingestion/pin)
- [Copy Data from S3 Bucket to public storage](https://docs.bacalhau.org/data-ingestion/s3)

:::info
You can mount your data anywhere on your machine, and Bacalhau will be able to run against that data
:::

### Step 6 - Run the Workload on Bacalhau

To run your workload, run the following command:

```shell
$ bacalhau docker run --input ipfs://${CID} ${IMAGE} ${CMD}
```
To check the status of your job, run the following command:

```shell
$ bacalhau list 
```
To get more information on your job

```shell
$ bacalhau describe JOB_ID
```

To download your job

```shell
$ bacalhau get JOB_ID
```

For example, running:

```shell
$ job_id=$(bacalhau docker run ubuntu echo hello)
$ bacalhau list --id-filter $job_id
$ sleep 5
$ bacalhau list --id-filter $job_id
$ bacalhau get $job_id
$ ls shards
```

outputs:

```
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
The `--input` flag does not support CID subpaths for `ipfs://` content.
:::

Alternatively, you can run your workload with a publicly accessible http(s) URL, which will download the data temporarily into your public storage:

```shell
$ export URL=https://download.geofabrik.de/antarctica-latest.osm.pbf
$ bacalhau docker run --input ${URL} ${IMAGE} ${CMD}

$ bacalhau list 

$ bacalhau get JOB_ID
```

:::caution
The `--input` flag does not support URL directories.
:::


<!-- <ReactPlayer playing controls url='https://www.youtube.com/watch?v=t2AHD8yJhLY' playing='false'/> -->

## Troubleshooting
If you run into this compute error while running your docker image 

```
Creating job for submission ... done ✅
Finding node(s) for the job ... done ✅
Node accepted the job ... done ✅
Error while executing the job.
```

This can often be resolved by re-tagging your docker image

## Support

If have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://join.slack.com/t/bacalhauproject/shared_invite/zt-1sihp4vxf-TjkbXz6JRQpg2AhetPzYYQ/archives/C02RLM3JHUY)(#bacalhau channel)
