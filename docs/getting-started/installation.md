---
sidebar_label: 'Installation'
sidebar_position: 1
---

# Getting Started with Bacalhau

In this tutorial, you'll learn how to install and run a job with the Bacalhau client. 

## Install the Bacalhau client

You can install or update the Bacalhau CLI by running the following command in a terminal:

```
curl -sL https://get.bacalhau.org/install.sh | bash
```


:::tip

Windows users can download the [latest release tarball from Github](https://github.com/filecoin-project/bacalhau/releases) and extract `bacalhau.exe` to anywhere that is on the PATH.

:::

Once your Bacalhau client is installed, it will show the client and server version. Your client and server versions must be aligned before you can run a job with Bacalhau client. You can use the code below to check this:

```
❯ bacalhau version
Client Version: v0.x.y
Server Version: v0.x.y
```

If you're wondering which server is being used, the Bacalhau Project has a [public Bacalhau server network](../about-bacalhau/introduction) that's shared with the community. This server allows you to launch your jobs from your computer without maintaining a compute cluster on your own.

Going further, we will look at some commands to run a simple job. For a complete overview of the `bacalhau` commands, take a look at the [CLI Reference page](../all-flags).

## Submit a "Hello World" job

The easiest way to submit a job is using the `docker run` verb. Let's take a quick look at its syntax: 

`bacalhau docker run [FLAGS] IMAGE[:TAG] [COMMAND]` 

While the command is designed to resemble Docker's run command which you may be familiar with, Bacalhau introduces a whole new set of [available flags (see CLI Reference)](../all-flags#docker-run) to support its computing model.

The code snippet below submits a job that runs an `echo` program within an [Ubuntu container](https://hub.docker.com/_/ubuntu). When a job is sumbitted, Bacalhau prints out the related job id:

```zsh
❯ bacalhau docker run ubuntu echo Hello World
3b39baee-5714-4f17-aa71-1f5824665ad6
```

The job id above is shown in its full form. For convenience, you can use the shortened version, in this case: `3b39baee`. We will store that portion of the job id in an environment variable so that we can reuse it later on.

After the above command is run, a job is submitted to the public network, which processes the job as described in the [Job Lifecycle page](../about-bacalhau/architecture#job-lifecycle). To check the current job's state, we can use the `list` verb as shown below.

```
❯ export JOB_ID=3b39baee # make sure to use the right job id from the docker run command

❯ bacalhau list --id-filter=${JOB_ID}
 CREATED   ID        JOB                      STATE      VERIFIED  PUBLISHED
 07:20:32  3b39baee  Docker ubuntu echo H...  Published            /ipfs/bafybeidu4zm6w...
```

A `Published/Completed` state indicates the job has completed successfully and the results are stored in the IPFS location under the `PUBLISHED` column.  

For a comprehensive list of flags you can pass to the list command check out [the related CLI Reference page](../all-flags#list).


## Get results

After the job has finished processing, its outputs are stored on IPFS. To download outputs locally, we can use the `get` verb.

First, we'll create and move into a directory that will store our job outputs. Next, we use the `get` verb to download the job outputs into the current directory.


```
❯ mkdir -p /tmp/myfolder
❯ cd /tmp/myfolder

❯ bacalhau get ${JOB_ID}
15:44:12.278 | INF bacalhau/get.go:67 > Fetching results of job '3b39baee'...
15:44:18.463 | INF ipfs/downloader.go:115 > Found 1 result shards, downloading to temporary folder.
15:44:21.17 | INF ipfs/downloader.go:195 > Combining shard from output volume 'outputs' to final location: '/tmp/myfolder'
```

:::note

This command prints out a number of verbose logs- these are meant for Bacalhau developers. You can safely ignore them, per [issue #614](https://github.com/filecoin-project/bacalhau/issues/614))

:::

At this point, the outputs have been downloaded locally and we are ready to inspect them. Each job creates 3 useful artifacts: the `stdout` and `stderr` files, as well as a `volumes/` directory. 

For the scope this of this guide, we will only look at the `stdout` file, but in a real world scenario, you should also look at output data stored within the `volumes/` directory. The `shards/` folder can be ignored.

```
❯ ls -l
total 8
drwxr-xr-x  3 enricorotundo  wheel  96 Sep 13 15:58 shards
-rw-r--r--  1 enricorotundo  wheel   0 Sep 13 15:58 stderr
-rw-r--r--  1 enricorotundo  wheel  12 Sep 13 15:58 stdout
drwxr-xr-x  3 enricorotundo  wheel  96 Sep 13 15:58 volumes
```

We submitted a job to print a string message to a [standard output](https://en.wikipedia.org/wiki/Standard_streams), so we have enough to now inspect the content of the related file:

```
❯ cat /tmp/myfolder/stdout
Hello World
```

With that, you have just sucessfully run a job on the Bacalhau network! :fish:

## Where to go next?

Here are a few resources that provides a deeper dive into running jobs with Bacalhau: 

* [How to run an existing workload on Bacalhau](../getting-started/docker-workload-onboarding.md)
* [Walk through a more data intensive demo](../examples/data-engineering/image-processing/index.md)
* [Check out the Bacalhau CLI Reference page](../all-flags.md)

## Support

For help with the Bacalhau client, installation, or other questions, please reach out to the [Bacalhau team via Slack (#bacalhau channel)](https://filecoin.io/slack).
