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

:::info
By default, you will submit to the Bacalhau public network, but the same CLI can be configured to submit to a private Bacalhau network. For more information, please read Running [Bacalhau on a Private Network](../setting-up/networking-instructions/private-cluster).
:::

### Step 1.1 - Install the Bacalhau CLI


<Tabs
defaultValue="Linux/macOS(CLI)"
values={[
{label: 'Linux/macOS(CLI)', value: 'Linux/macOS(CLI)'},
{label: 'Windows(CLI)', value: 'Windows(CLI)'},
{label: 'Docker', value: 'Docker'},
]}>
<TabItem value="Linux/macOS(CLI)">

    You can install or update the Bacalhau CLI by running the commands in a terminal. You may need sudo mode or root password to install the local Bacalhau binary to `/usr/local/bin`:
    ```shell
    curl -sL https://get.bacalhau.org/install.sh | bash
    ```

</TabItem>
<TabItem value="Windows(CLI)">

    Windows users can download the [latest release tarball from Github](https://github.com/bacalhau-project/bacalhau/releases) and extract `bacalhau.exe` to any location available in the PATH environment variable.

</TabItem>
<TabItem value="Docker">

    ```shell
    docker image rm -f ghcr.io/bacalhau-project/bacalhau:latest # Remove old image if it exists
    docker pull ghcr.io/bacalhau-project/bacalhau:latest
    ```

    To run a specific version of Bacalhau using Docker, use the command docker run -it ghcr.io/bacalhau-project/bacalhau:v1.0.3, where "v1.0.3" is the version you want to run; note that the "latest" tag will not re-download the image if you have an older version. For more information on running the Docker image, check out the [Bacalhau docker image example](../setting-up/workload-onboarding/bacalhau-docker-image/index.md).
</TabItem>
</Tabs>


### Step 1.2 - Verify the Installation

To verify installation and check the version of the client and server, use the `version` command.
To run a Bacalhau client command with Docker, prefix it with `docker run ghcr.io/bacalhau-project/bacalhau:latest`.

<Tabs
defaultValue="Linux/macOS/Windows(CLI)"
values={[
{label: 'Linux/macOS/Windows(CLI)', value: 'Linux/macOS/Windows(CLI)'},
{label: 'Docker', value: 'Docker'},
]}>
<TabItem value="Linux/macOS/Windows(CLI)">

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

If you're wondering which server is being used, the Bacalhau Project has a [public Bacalhau server network](../setting-up/running-node/quick-start.md#ensure-your-storage-server-is-running) that's shared with the community. This server allows you to launch your jobs from your computer without maintaining a compute cluster on your own.


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

    ```shell
    Job successfully submitted. Job ID: f8e7789d-8e76-4e6c-8e71-436e2d76c72e
    Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):

        Communicating with the network  ................  done ✅  0.2s
           Creating job for submission  ................  done ✅  0.7s
                       Job in progress  ................  done ✅  2.1s

    To download the results, execute:
        bacalhau get f8e7789d-8e76-4e6c-8e71-436e2d76c72e

    To get more details about the run, execute:
        bacalhau describe f8e7789d-8e76-4e6c-8e71-436e2d76c72e
    ```

After the above command is run, the job is submitted to the public network, which processes the job and Bacalhau prints out the related job id:

```
Job successfully submitted. Job ID: 9d20bbad-c3fc-48f8-907b-1da61c927fbd
Checking job status...
```

The `job_id` above is shown in its full form. For convenience, you can use the shortened version, in this case: `9d20bbad`.

:::info
While this command is designed to resemble Docker's run command which you may be familiar with, Bacalhau introduces a whole new set of [flags](../dev/cli-reference/all-flags.md#docker-run) to support its computing model.
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

```shell
14:02:25.992 | INF pkg/repo/fs.go:81 > Initializing repo at '/root/.bacalhau' for environment 'production'
19b105c9-4cb5-43bd-a12f-d715d738addd
```

</TabItem>
</Tabs>


## Step 3 - Checking the State of your Jobs
After having deployed the job, we now can use the CLI for the interaction with the network. The jobs were sent to the public demo network, where it was processed and we can call the following functions. The `job_id` will differ for every submission.
### Step 3.1 - Job status:

You can check the status of the job using `bacalhau list` command adding the `--id-filter` flag and specifying your job id.


```shell
bacalhau list --id-filter 9d20bbad
```
Let's take a look at the results of the command execution in the terminal:

```shell
CREATED   ID          JOB                                       STATE      PUBLISHED
15:24:31  0ed7617d    Type:"docker",Params:"map[Entrypoint:<ni  Completed
                       l> EnvironmentVariables:[] Image:ubuntu:
                       latest Parameters:[sh -c uname -a && ech
                       o "Hello from Docker Bacalhau!"] Working
                       Directory:]"
```

When it says `Completed`, that means the job is done, and we can get the results.

:::info
For a comprehensive list of flags you can pass to the list command check out [the related CLI Reference page](../dev/cli-reference/all-flags.md#list-2).
:::

### Step 3.2 - Job information:

You can find out more information about your job by using `bacalhau describe`.

```shell
bacalhau describe 9d20bbad
```
Let's take a look at the results of the command execution in the terminal:

```shell

    Job:
        APIVersion: V1beta2
        Metadata:
            ClientID: 0ff57b2521334a92e9ddab4b2f8202c887b1eaa35d2aa945ab0e247d3bc0aa88
            CreatedAt: "2023-12-21T15:24:31.750306239Z"
            ID: 0ed7617d-d5ff-40f7-8411-89830b3f3058
            Requester:
            RequesterNodeID: QmbxGSsM6saCTyKkiWSxhJCt6Fgj7M9cns1vzYtfDbB5Ws
            RequesterPublicKey: CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDEHTUAD1JzO0130W9vsaDGhU0PVgpcNjG3fYlE0sJ1BiBWENFuP4jx3Q9alcjNGhdRFdju0Mb/fidTOtJcPhxTdb+H6JxFP6HsADGes9jU4ylBU2SL2vfdb0KXzKdXjNHGGf4BuCGTcH07Oqxp209diK/cT7takL2fLjcgs1tM+6PzlfGzFqCPxvh9Sa0ek34mdmHjcp1XH8yjF1OKOuHvD+pYphqvOBL/2LEN+EBC4fz/QUnhUajCmKYO83MJcNUXSGxb4AN6K3DpVV+cJph7fj9ADdP7i996o2S4Gkz8W4Wpt/jICaPpkUjmyU3Jgcw7MHkZaYEzWxnnO2J936+pAgMBAAE=
        Spec:
        ...

```

This outputs all information about the job, including stdout, stderr, where the job was scheduled, and so on.

### Step 3.3 - Job download:

You can download your job results directly by using `bacalhau get`.


```shell
bacalhau get 9d20bbad
```
This results in

```shell
Fetching results of job '0ed7617d'...
Results for job '0ed7617d' have been written to...
/Users/test/job-0ed7617d
```

In the command below, we created a directory called `myfolder` and download our job output to be stored in that directory.

```shell
Fetching results of job '0ed7617d'...
Results for job '0ed7617d' have been written to...
/myfolder
```

:::info
While executing this command, you may encounter warnings regarding receive and send buffer sizes: `failed to sufficiently increase receive buffer size`. These warnings can arise due to limitations in the UDP buffer used by Bacalhau to process tasks. Additional information can be found in [https://github.com/quic-go/quic-go/wiki/UDP-Buffer-Sizes](https://github.com/quic-go/quic-go/wiki/UDP-Buffer-Sizes).
:::

After the download has finished you should see the following contents in the results directory.

```shell
job-0ed7617d
├── exitCode
├── outputs
├── stderr
└── stdout
```

## Step 4 - Viewing your Job Output

```shell
$ cat job-9d20bbad/stdout
```

That should print out the string `Hello World`.

```shell
Hello world
```

With that, you have just successfully run a job on Bacalhau! :fish:

## Step 5 - Where to go next?

Here are few resources that provide a deeper dive into running jobs with Bacalhau:

 [How Bacalhau works](../getting-started/architecture.md)
 [Setting up Bacalhau](../setting-up)
 [Examples & Use Cases](../examples)


## Support

If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).
