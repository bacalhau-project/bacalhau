---
sidebar_label: 'Installation'
sidebar_position: 2
---

# Getting Started with Bacalhau

In this tutorial, you'll learn how to install and run a job with the Bacalhau client.

## Install the Bacalhau Client

The Bacalhau client is a command-line interface (CLI) that allows you to submit jobs to the Bacalhau network. The Bacalhau client is available for Linux, macOS, and Windows. you can also run Bacalhau in a Docker container. 

If you don't already have Docker installed, see the [full example](../examples/workload-onboarding/bacalhau-docker-image/index.md) for installing and running Bacalhau on Docker.

### Install the Bacalhau CLI 

You can install or update the Bacalhau CLI or pull a Docker image by running the following command in a terminal.
You may need sudo mode or root password to install the local Bacalhau binary to `/usr/local/bin`:

:::tip
Windows users can download the [latest release tarball from Github](https://github.com/bacalhau-project/bacalhau/releases) and extract `bacalhau.exe` to anywhere on the PATH.
:::


import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs
    defaultValue="CLI"
    values={[
        {label: 'CLI', value: 'CLI'},
        {label: 'Docker', value: 'Docker'},
    ]}>
<TabItem value="CLI">

    curl -sL https://get.bacalhau.org/install.sh | bash

</TabItem>
<TabItem value="Docker">

    docker image rm -f ghcr.io/bacalhau-project/bacalhau:latest # Remove old image if it exists
    docker pull ghcr.io/bacalhau-project/bacalhau:latest

</TabItem>
</Tabs>

_ See the [full example](../examples/workload-onboarding/bacalhau-docker-image/index.md) for installing and running bacalhau on Docker. _

:::info
The "latest" tag signifies a specific Bacalhau [release](https://github.com/bacalhau-project/bacalhau/releases) or Docker image. If your machine has an older "latest" version, it won't re-download. To run a particular version, use: `docker run -it ghcr.io/bacalhau-project/bacalhau:v0.3.23` (for version 0.3.23).
:::

### Verify the Installation

To run and Bacalhau client command with Docker, prefix it with `docker run ghcr.io/bacalhau-project/bacalhau:latest`. 

To verify installation and check the version of the client and server, use the `version` command, you can run the command:


<Tabs
    defaultValue="CLI"
    values={[
        {label: 'CLI', value: 'CLI'},
        {label: 'Docker', value: 'Docker'},
    ]}>
<TabItem value="CLI">

    bacalhau version

</TabItem>
<TabItem value="Docker">

    docker run -it ghcr.io/bacalhau-project/bacalhau:latest version

</TabItem>
</Tabs>

If you're wondering which server is being used, the Bacalhau Project has a [public Bacalhau server network](https://docs.bacalhau.org/#our-vision) that's shared with the community. This server allows you to launch your jobs from your computer without maintaining a compute cluster on your own.

Going further, we will look at some commands to run a simple job. For a complete overview of the `bacalhau` commands, take a look at the [CLI Reference page](../all-flags).

:::info
If you want to pass files between the Docker Bacalhau CLI and your desktop, don't forget to mount a volume. You can see an example of this in the [Bacalhau Docker example](../examples/workload-onboarding/bacalhau-docker-image/index.md).
:::

## Let's submit a "Hello World" Job

The easiest way to submit a job is using the `bacalhau docker run` command. Let's take a quick look at its syntax:

`bacalhau docker run [FLAGS] IMAGE[:TAG] [COMMAND]`

To run our Hello world job, the command below submits a job that runs an [echo](https://en.wikipedia.org/wiki/Echo_(command)) program within an [Ubuntu container](https://hub.docker.com/_/ubuntu):

```shell
$ bacalhau docker run ubuntu echo Hello World
```
:::info
While this command is designed to resemble Docker's run command which you may be familiar with, Bacalhau introduces a whole new set of [flags (see CLI Reference)](https://docs.bacalhau.org/all-flags#docker-run) to support its computing model.
:::

After the above command is run, the job is submitted to the public network, which processes the job and Bacalhau prints out the related job id:

```
Job successfully submitted. Job ID: 3b39baee-5714-4f17-aa71-1f5824665ad6
Checking job status...
```

The `job_id` above is shown in its full form. For convenience, you can use the shortened version, in this case: `3b39baee`. For ease, we store the `job_id` in an environment variable.

```shell
$ export JOB_ID=3b39baee # make sure to use the right job id from the docker run command
```

## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`. 


```bash
bacalhau list --id-filter ${JOB_ID}
```

When it says `Completed`, that means the job is done, and we can get the results.

```
 CREATED   ID        JOB                      STATE      VERIFIED  COMPLETED
 07:20:32  3b39baee  Docker ubuntu echo H...  Published            /ipfs/bafybeidu4zm6w...
```

:::info
For a comprehensive list of flags you can pass to the list command check out [the related CLI Reference page](../all-flags#list).
:::

- **Job information**: You can find out more information about your job by using `bacalhau describe`.

```bash
bacalhau describe ${JOB_ID}
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory called `myfolder` and download our job output to be stored in that directory.


```bash
$ mkdir -p /tmp/myfolder
$ cd /tmp/myfolder

bacalhau get $JOB_ID --output-dir results
```

After the download has finished you should see the following contents in results directory

## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**, **per_shard files**, and the **raw** directory. To view the file, run the following command:

```shell
$ cat /tmp/myfolder/job-id/combined_results/stdout
```

That should print out the string `Hello World`.

With that, you have just successfully run a job on the Bacalhau network! :fish:


## Where to go next?

Here are a few resources that provides a deeper dive into running jobs with Bacalhau:

* [How to run an existing workload on Bacalhau](../getting-started/docker-workload-onboarding.md)
* [Walk through a more data intensive demo](../examples/data-engineering/image-processing/index.md)
* [Check out the Bacalhau CLI Reference page](../all-flags.md)


## Need Support?

If have questions or need support or guidance, please reach out to the [Bacalhau team via Slack (#bacalhau channel)](https://filecoin.io/slack)
