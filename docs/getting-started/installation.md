---
sidebar_label: 'Installation'
sidebar_position: 1
---

# Getting Started with Bacalhau

In this tutorial, you'll learn how to install and run a job with the Bacalhau client. 

## Install the Bacalhau Client

The Bacalhau client is a command-line interface (CLI) that allows you to submit jobs to the Bacalhau network. The Bacalhau client is available for Linux, macOS, and Windows. It is also available as a Docker image.

### Installing the Bacalhau CLI Locally

You can install or update the Bacalhau CLI by running the following command in a terminal.
It will prompt for the root password to copy the Bacalhau binary to `/usr/local/bin`:

```bash
$ curl -sL https://get.bacalhau.org/install.sh | bash
```

:::tip
Windows users can download the [latest release tarball from Github](https://github.com/filecoin-project/bacalhau/releases) and extract `bacalhau.exe` to anywhere that is on the PATH.
:::

### Running The Bacalhau CLI Via Docker

Instead of installing the client, you can use the [Bacalhau Docker image](https://github.com/orgs/bacalhau-project/packages/container/package/bacalhau) to run the client. There's also a [comprehensive example](../examples/workload-onboarding/bacalhau-docker-image/index.md). To pull the latest image run the following command:

```shell
$ docker pull ghcr.io/bacalhau-project/bacalhau:latest
```

:::warning
Remember that the "latest" tag is just a string. It doesn't refer to the latest version of the Bacalhau client, it refers to an image that has the "latest" tag. Therefore, if your machine has already downloaded the "latest" image, it won't download it again. To force a download, you can use the `--no-cache` flag.
:::

Now you can run any Bacalhau client command by prefixing it with `docker run ghcr.io/bacalhau-project/bacalhau:latest`. For example, to run the `version` command, you can run:

```shell
$ docker run -it ghcr.io/bacalhau-project/bacalhau:latest version
```

:::tip
If you want to pass files between the Docker Bacalhau CLI and your desktop, don't forget to mount a volume. You can see an example of this in the [Bacalhau Docker example](../examples/workload-onboarding/bacalhau-docker-image/index.md).
:::

### Verify the Installation

Once your Bacalhau client is installed, it will show the client and server version. Your client and server versions must be aligned before you can run a job with Bacalhau client. You can use the code below to check this:

```shell
$ bacalhau version
```

If you're wondering which server is being used, the Bacalhau Project has a [public Bacalhau server network](https://docs.bacalhau.org/#our-vision) that's shared with the community. This server allows you to launch your jobs from your computer without maintaining a compute cluster on your own.

Going further, we will look at some commands to run a simple job. For a complete overview of the `bacalhau` commands, take a look at the [CLI Reference page](../all-flags).

## Submit a "Hello World" job

The easiest way to submit a job is using the `docker run` verb. Let's take a quick look at its syntax: 

`bacalhau docker run [FLAGS] IMAGE[:TAG] [COMMAND]` 

While the command is designed to resemble Docker's run command which you may be familiar with, Bacalhau introduces a whole new set of [available flags (see CLI Reference)](../all-flags#docker-run) to support its computing model.

The command below submits a job that runs an `echo` program within an [Ubuntu container](https://hub.docker.com/_/ubuntu):

```shell
$ bacalhau docker run ubuntu echo Hello World
```

When a job is sumbitted, Bacalhau prints out the related job id:

```
Job successfully submitted. Job ID: 3b39baee-5714-4f17-aa71-1f5824665ad6
Checking job status...
```

The job id above is shown in its full form. For convenience, you can use the shortened version, in this case: `3b39baee`. We will store that portion of the job id in an environment variable so that we can reuse it later on.

After the above command is run, a job is submitted to the public network, which processes the job as described in the [Job Lifecycle page](../about-bacalhau/architecture#job-lifecycle). To check the current job's state, we can use the `list` verb as shown below.

```shell
$ export JOB_ID=3b39baee # make sure to use the right job id from the docker run command

$ bacalhau list --id-filter=${JOB_ID}
```

The list command prints out the following text:

```
 CREATED   ID        JOB                      STATE      VERIFIED  COMPLETED
 07:20:32  3b39baee  Docker ubuntu echo H...  Published            /ipfs/bafybeidu4zm6w...
```

A `Completed` state indicates the job has completed successfully and the results are stored in the IPFS location under the `PUBLISHED` column.  

For a comprehensive list of flags you can pass to the list command check out [the related CLI Reference page](../all-flags#list).


## Get results

After the job has finished processing, its outputs are stored on IPFS. To download outputs locally.

First, we'll create a directory that will store our job outputs.

```shell
$ mkdir -p /tmp/myfolder
$ cd /tmp/myfolder
```

Next, we use the `get` verb to download the job outputs into the current directory.

```shell
$ bacalhau get ${JOB_ID}
```


:::info

The `get` command may be slow at times, please be patient or retry upon failure.

:::

At this point, the outputs have been downloaded locally and we are ready to inspect them. Each job creates 3 subfolders: the *combined_results*, *per_shard* files, and the *raw* directory. In each of these sub_folders, you'll find the *stdout* and *stderr* file.

For the scope this of this guide, we will only look at the **stdout** file. To inspect the content of the file, use the code below:

```shell
$ cat /tmp/myfolder/job-id/combined_results/stdout
```

That should print out the string `Hello World`.
With that, you have just sucessfully run a job on the Bacalhau network! :fish:

## Where to go next?

Here are a few resources that provides a deeper dive into running jobs with Bacalhau: 

* [How to run an existing workload on Bacalhau](../getting-started/docker-workload-onboarding.md)
* [Walk through a more data intensive demo](../examples/data-engineering/image-processing/index.md)
* [Check out the Bacalhau CLI Reference page](../all-flags.md)


## Need Support?

If have questions or need support or guidance, please reach out to the [Bacalhau team via Slack (#bacalhau channel)](https://filecoin.io/slack)
