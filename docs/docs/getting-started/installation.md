---
sidebar_label: 'Installation'
sidebar_position: 0
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# Installation

In this tutorial, you'll learn how to install and run a job with the Bacalhau client using the Bacalhau CLI or Docker.

## Step 1 - Install the Bacalhau Client

The Bacalhau client is a command-line interface (CLI) that allows you to submit jobs to the Bacalhau.  The Bacalhau client is available for Linux, macOS, and Windows. You can also run the Bacalhau client in a Docker container.

:::tip
By default, you will submit to the Bacalhau public network, but the same CLI can be configured to submit to a private Bacalhau network. For more information, please read Running [Bacalhau on a Private Network](../next-steps/private-cluster).
:::

### Step 1.1 - Install the Bacalhau CLI


<Tabs
defaultValue="Linux/macOS"
values={[
{label: 'Linux/macOS', value: 'Linux/macOS'},
{label: 'Windows', value: 'Windows'},
{label: 'Docker', value: 'Docker'},
]}>
<TabItem value="Linux/macOS">

    You can install or update the Bacalhau CLI by running the commands in a terminal. You may need sudo mode or root password to install the local Bacalhau binary to `/usr/local/bin`:
    ```shell
    curl -sL https://get.bacalhau.org/install.sh | bash
    ```

</TabItem>
<TabItem value="Windows">

    Windows users can download the [latest release tarball from Github](https://github.com/bacalhau-project/bacalhau/releases) and extract `bacalhau.exe` to any location available in the PATH environment variable.

</TabItem>
<TabItem value="Docker">

    ```shell
    docker image rm -f ghcr.io/bacalhau-project/bacalhau:latest # Remove old image if it exists
    docker pull ghcr.io/bacalhau-project/bacalhau:latest
    ```
    
    To run a specific version of Bacalhau using Docker, use the command docker run -it ghcr.io/bacalhau-project/bacalhau:v1.0.3, where "v1.0.3" is the version you want to run; note that the "latest" tag will not re-download the image if you have an older version. For more information on running the Docker image, check out the [Bacalhau docker image example](../examples/workload-onboarding/bacalhau-docker-image/index.md).
</TabItem>
</Tabs>


### Step 1.2 - Verify the Installation

To verify installation and check the version of the client and server, use the `version` command.
To run a Bacalhau client command with Docker, prefix it with `docker run ghcr.io/bacalhau-project/bacalhau:latest`.

<Tabs
defaultValue="Linux/macOS/Windows"
values={[
{label: 'Linux/macOS/Windows', value: 'Linux/macOS/Windows'},
{label: 'Docker', value: 'Docker'},
]}>
<TabItem value="Linux/macOS/Windows">

    ```shell
    bacalhau version
    ```

</TabItem>
<TabItem value="Docker">

    ```shell
    docker run -it ghcr.io/bacalhau-project/bacalhau:latest version
    ```

</TabItem>
</Tabs>

If you're wondering which server is being used, the Bacalhau Project has a [public Bacalhau server network](https://docs.bacalhau.org/#our-vision) that's shared with the community. This server allows you to launch your jobs from your computer without maintaining a compute cluster on your own.


## Step 2 - Submit a Hello World job

To submit a job in Bacalhau, we will use the `bacalhau docker run` command. The command runs a job using the Docker executor on the node. Let's take a quick look at its syntax:

```shell
bacalhau docker run [FLAGS] IMAGE[:TAG] [COMMAND]
```

<Tabs
defaultValue="CLI"
values={[
{label: 'CLI', value: 'CLI'},
{label: 'Docker', value: 'Docker'},
]}>
<TabItem value="CLI">

    ```shell
    bacalhau docker run ubuntu echo Hello World
    ```

    We will use the command to submit a Hello World job that runs an [echo](https://en.wikipedia.org/wiki/Echo_(command)) program within an [Ubuntu container](https://hub.docker.com/_/ubuntu).

    Let's take a look at the results of the command execution in the terminal: 

![image](../../static/img/Installation/bacalhau-docker-run1.png 'bacalhau-docker-run1')

After the above command is run, the job is submitted to the public network, which processes the job and Bacalhau prints out the related job id:

```
Job successfully submitted. Job ID: 9d20bbad-c3fc-48f8-907b-1da61c927fbd
Checking job status...
```

The `job_id` above is shown in its full form. For convenience, you can use the shortened version, in this case: `9d20bbad`. 

:::info
While this command is designed to resemble Docker's run command which you may be familiar with, Bacalhau introduces a whole new set of [flags (see CLI Reference)](https://docs.bacalhau.org/all-flags#docker-run) to support its computing model.
:::

</TabItem>
<TabItem value="Docker">
```shell
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \ 
docker run \  
--id-only \  
--wait \  
ubuntu:latest -- \ 
sh -c 'uname -a && echo "Hello from Docker Bacalhau!"'
```

Let's take a look at the results of the command execution in the terminal:

![image](../../static/img/Installation/docker-run1.png 'docker-run')

</TabItem>
</Tabs>


## Step 3 - Checking the State of your Jobs

**Job status**: You can check the status of the job using `bacalhau list` command adding the `--id-filter` flag and specifying your job id.


```shell
bacalhau list --id-filter 9d20bbad
```
Let's take a look at the results of the command execution in the terminal: 

![image](../../static/img/Installation/bacalhau-list1.png 'bacalhau-list')

When it says `Completed`, that means the job is done, and we can get the results.

:::info
For a comprehensive list of flags you can pass to the list command check out [the related CLI Reference page](../all-flags#list).
:::

**Job information**: You can find out more information about your job by using `bacalhau describe`.

```shell
bacalhau describe 9d20bbad
```
Let's take a look at the results of the command execution in the terminal: 

![image](../../static/img/Installation/bacalhau-describe1.png 'bacalhau-describe')

This outputs all information about the job, including stdout, stderr, where the job was scheduled, and so on.

**Job download**: You can download your job results directly by using `bacalhau get`. 


```shell
bacalhau get 9d20bbad
```

![image](../../static/img/Installation/bacalhau-get-jobid.png 'bacalhau-get')

In the command below, we created a directory called `myfolder` and download our job output to be stored in that directory.

![image](../../static/img/Installation/bacalhau-get-myfolder1.png 'bacalhau-get')

:::info
While executing this command, you may encounter warnings regarding receive and send buffer sizes: `failed to sufficiently increase receive buffer size`. These warnings can arise due to limitations in the UDP buffer used by Bacalhau to process tasks. Additional information can be found in [https://github.com/quic-go/quic-go/wiki/UDP-Buffer-Sizes](https://github.com/quic-go/quic-go/wiki/UDP-Buffer-Sizes). 
:::

After the download has finished you should see the following contents in the results directory.

![image](../../static/img/Installation/tree-jobid1.png 'tree-jobid')


## Step 4 - Viewing your Job Output

```shell
$ cat job-9d20bbad/stdout
```

That should print out the string `Hello World`.

![image](../../static/img/Installation/cat-jobid1.png 'cat-jobid')

With that, you have just successfully run a job on the Bacalhau network! :fish:

## Step 5 - Where to go next?

Here are few resources that provide a deeper dive into running jobs with Bacalhau:

 [How Bacalhau works]  
 [Setting up Bacalhau]  
 [Examples & Use Cases]  


## Need Support?

If have questions or need support or guidance, please reach out to the [Bacalhau team via Slack (#bacalhau channel)](https://bit.ly/bacalhau-project-slack)