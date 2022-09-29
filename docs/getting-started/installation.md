---
sidebar_label: 'Installation' sidebar_position: 1
---

# Getting Started with Bacalhau

## Prerequisite: install Bacalhau client

Install (or update) the Bacalhau CLI by running the following command in a terminal.

```
curl -sL https://get.bacalhau.org/install.sh | bash
```

Windows users can download the [latest release tarball from Github](https://github.com/filecoin-project/bacalhau/releases) and extract `bacalhau.exe` to anywhere that is on the PATH.

Please make sure the Bacalhau client & server versions are aligned.

```
❯ bacalhau version
Client Version: v0.x.y
Server Version: v0.x.y
```

At this point you may be wondering what server is being used here.
That is a fair question because you have just installed a client.
The good news is the Bacalhau Project has made available a [public Bacalhau server network](../about-bacalhau/introduction) to the community!
This way you can just launch your jobs from your laptop without worrying about maintaing whole a compute cluster.

In this guide we provide all commands needed to get you started, but for a complete overview of the `bacalhau` command you can take a look at the [CLI Reference](../all-flags) page.

Now you are good to go!

## Submit a "Hello World" job

The easiest way to submit a job is using the `docker run` verb.
Let us break down its syntax first: `bacalhau docker run [FLAGS] IMAGE[:TAG] [COMMAND]`; while it is designed to resemble Docker's run command you are probably familiar with, Bacalhau introduces a whole new set of [available flags (see CLI Reference)](../all-flags#docker-run) to support its computing model.

The snipped below sumbits a job that runs an `echo` program within an [Ubuntu container](https://hub.docker.com/_/ubuntu).
When a job is sumbitted, Bacalhau prints out the related job id.

```zsh
❯ bacalhau docker run ubuntu echo Hello World
3b39baee-5714-4f17-aa71-1f5824665ad6
```

The job id above is shown in its full form.
For convenience, we can use the its shortened version consisting of the first part, in this case is `3b39baee`.
We will store that in an environment variable so that we can reuse it later on.

The job has now been sumbitted to the public network who is going to process it as described in the [Job Lifecycle page](../about-bacalhau/architecture#job-lifecycle).
To check the current job's state we can use the `list` verb below.
A `Published/Completed` state indicates the job has completed successfully and the results are stored in the IPFS location under the `PUBLISHED` column.
For a comprehensive list of flags you can pass to the list command check out [the related CLI Reference page](../all-flags#list).

```
❯ export JOB_ID=3b39baee # make sure to use the right job id from the docker run command

❯ bacalhau list --id-filter=${JOB_ID}
 CREATED   ID        JOB                      STATE      VERIFIED  PUBLISHED
 07:20:32  3b39baee  Docker ubuntu echo H...  Published            /ipfs/bafybeidu4zm6w...
```

## Get results

The job's outputs are now stored on IPFS, to download them locally we will use the `get` verb.

We achieve that by running the commands in the snippet below.
First, let us create and move into a directory that will store our job outputs.
Second, use the `get` verb to download the job outputs into the current directory.
_This command prints out a number of verbose logs, although these meant for Bacalhau developers you may want to ignore them (this will soon go away: [issue #614](https://github.com/filecoin-project/bacalhau/issues/614))._

```
❯ mkdir -p /tmp/myfolder
❯ cd /tmp/myfolder

❯ bacalhau get ${JOB_ID}
15:44:12.278 | INF bacalhau/get.go:67 > Fetching results of job '3b39baee'...
15:44:18.463 | INF ipfs/downloader.go:115 > Found 1 result shards, downloading to temporary folder.
15:44:21.17 | INF ipfs/downloader.go:195 > Combining shard from output volume 'outputs' to final location: '/tmp/myfolder'
```

At this point the outputs have been downloaded locally and we are ready to inspect them, but what do the outputs consist of?
Each job creates 3 useful artifacts: the `stdout` and `stderr` files as well as a `volumes/` directory.
For the scope this of this guide we will only look at the `stdout` file, but in a real world scenario you would be looking at output data stored within the `volumes/` directory.
The `shards/` folder can be ignored.

```
❯ ls -l
total 8
drwxr-xr-x  3 enricorotundo  wheel  96 Sep 13 15:58 shards
-rw-r--r--  1 enricorotundo  wheel   0 Sep 13 15:58 stderr
-rw-r--r--  1 enricorotundo  wheel  12 Sep 13 15:58 stdout
drwxr-xr-x  3 enricorotundo  wheel  96 Sep 13 15:58 volumes
```

We had submitted a simple job to print a string message to [standard output](https://en.wikipedia.org/wiki/Standard_streams), therefore it is enough to inspect the content of the related file.

```
❯ cat /tmp/myfolder/stdout
Hello World
```

Hooray, you have just sucessfully run a job on the Bacalhau network! :fish:

## Where to go next?

* [How to run an existing workload on Bacalhau](../getting-started/workload-onboarding.md)
* [Walk through a more data intensive demo](../examples/data-engineering/image-processing/index.md)
* [Check out the Bacalhau CLI Reference page](../all-flags.md)

## Support

Please reach out to the [Bacalhau team via Slack](https://filecoinproject.slack.com/archives/C02RLM3JHUY) to seek help or in case of any issues.
