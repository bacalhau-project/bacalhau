---
sidebar_label: 'Installation'
sidebar_position: 2
---

# Getting Started with Bacalhau

In this tutorial, you'll learn how to install and run a job with the Bacalhau client using the Bacalhau CLI or Docker.

## The Bacalhau Client

The Bacalhau client is a command-line interface (CLI) that allows you to submit jobs to the Bacalhau.  The Bacalhau client is available for Linux, macOS, and Windows. You can also run the Bacalhau client in a Docker container.

:::tip
By default, you will submit to the Bacalhau public network, but the same CLI can be configured to submit to a private Bacalhau network. For more information, please read Running [Bacalhau on a Private Network](../next-steps/private-cluster).
:::

### Install the Bacalhau CLI

You can install or update the Bacalhau CLI or pull a Docker image by running the commands in a terminal.
You may need sudo mode or root password to install the local Bacalhau binary to `/usr/local/bin`:

:::tip
Using the **CLI**: Windows users can download the [latest release tarball from Github](https://github.com/bacalhau-project/bacalhau/releases) and extract `bacalhau.exe` to anywhere on the PATH.
:::

:::info
To run a specific version of Bacalhau using Docker, use the command docker run -it ghcr.io/bacalhau-project/bacalhau:v1.0.3, where "v1.0.3" is the version you want to run; note that the "latest" tag will not re-download the image if you have an older version. For more information on running the Docker image, check out the [Bacalhau docker image example](../examples/workload-onboarding/bacalhau-docker-image/index.md).
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


## Let's submit a Hello World job

To submit a job in Bacalhau, we will use the `bacalhau docker run` command. Let's take a quick look at its syntax:

`bacalhau docker run [FLAGS] IMAGE[:TAG] [COMMAND]`

The command below submits a Hello World job that runs an [echo](https://en.wikipedia.org/wiki/Echo_(command)) program within an [Ubuntu container](https://hub.docker.com/_/ubuntu):

<Tabs
defaultValue="CLI"
values={[
{label: 'CLI', value: 'CLI'},
{label: 'Docker', value: 'Docker'},
]}>
<TabItem value="CLI">

    bacalhau docker run ubuntu echo Hello World

</TabItem>
<TabItem value="Docker">

    docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    docker run \
        --id-only \
        --wait \
        ubuntu:latest -- \
            sh -c 'uname -a && echo "Hello from Docker Bacalhau!"'

</TabItem>
</Tabs>

:::info
While this command is designed to resemble Docker's run command which you may be familiar with, Bacalhau introduces a whole new set of [flags (see CLI Reference)](https://docs.bacalhau.org/all-flags#docker-run) to support its computing model.
:::

After the above command is run, the job is submitted to the public network, which processes the job and Bacalhau prints out the related job id:

```
Job successfully submitted. Job ID: 3b39baee-5714-4f17-aa71-1f5824665ad6
Checking job status...
```

The `job_id` above is shown in its full form. For convenience, you can use the shortened version, in this case: `3b39baee`. 

## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`.


```shell
bacalhau list --id-filter 3b39baee
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

```shell
bacalhau describe 3b39baee
```

This outputs all information about the job, including stdout, stderr, where the job was scheduled, and so on.

- **Job download**: You can download your job results directly by using `bacalhau get`. In the command below, we created a directory called `myfolder` and download our job output to be stored in that directory.


```shell
bacalhau get 3b39baee
```

After the download has finished you should see the following contents in the results directory.

```shell
$ tree job-3b39baee
job-3b39baee
├── exitCode
├── outputs
├── stderr
└── stdout
```

## Viewing your Job Output

```shell
$ cat job-3b39baee/stdout
```

That should print out the string `Hello World`.

With that, you have just successfully run a job on the Bacalhau network! :fish:

## Where to go next?

Here are a few resources that provide a deeper dive into running jobs with Bacalhau:

* [How to run an existing workload on Bacalhau](../getting-started/docker-workload-onboarding.md)
* [Walk through a more data intensive demo](../examples/data-engineering/image-processing/index.md)
* [Check out the Bacalhau CLI Reference page](../all-flags.md)


## Need Support?

If have questions or need support or guidance, please reach out to the [Bacalhau team via Slack (#bacalhau channel)](https://bit.ly/bacalhau-project-slack)
