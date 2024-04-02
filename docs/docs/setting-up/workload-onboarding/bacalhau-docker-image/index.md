---
sidebar_label: "Bacalhau Docker Image"
sidebar_position: 1
description: How to use the Bacalhau Docker image
---
# Bacalhau Docker Image

This documentation explains how to use the Bacalhau Docker image to run tasks and manage them using the Bacalhau client.

## Prerequisites

To get started, you need to install the Bacalhau client (see more information [here](../../../getting-started/installation.md)) and Docker.

## 1. Pull the Bacalhau Docker image

The first step is to pull the Bacalhau Docker image from the [Github container registry](https://github.com/orgs/bacalhau-project/packages/container/package/bacalhau).

```shell
docker pull ghcr.io/bacalhau-project/bacalhau:latest

Expected output:
latest: Pulling from bacalhau-project/bacalhau
d14ccdd25413: Pull complete
621f190d05c8: Pull complete
Digest: sha256:3cda5619984de9b56c738c50f94188684170f54f7e417f8dcbe74ff8ec8eb434
Status: Downloaded newer image for ghcr.io/bacalhau-project/bacalhau:latest
ghcr.io/bacalhau-project/bacalhau:latest
```

You can also pull a specific version of the image, e.g.:

```bash
docker pull ghcr.io/bacalhau-project/bacalhau:v0.3.16
```

:::warning
Remember that the "latest" tag is just a string. It doesn't refer to the latest version of the Bacalhau client, it refers to an image that has the "latest" tag. Therefore, if your machine has already downloaded the "latest" image, it won't download it again. To force a download, you can use the `--no-cache` flag.
:::

## 2. Check version

To check the version of the Bacalhau client, run:

```shell
docker run -t ghcr.io/bacalhau-project/bacalhau:latest version

Expected Output:
13:38:54.518 | INF pkg/repo/fs.go:81 > Initializing repo at '/root/.bacalhau' for environment 'production'
CLIENT  SERVER  UPDATE MESSAGE
v1.2.0  v1.2.0
```

## 3. Running a Bacalhau Job

In the example below, an Ubuntu-based job runs to print the message 'Hello from Docker Bacalhau:

```shell
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    docker run \
        --id-only \
        --wait \
        ubuntu:latest \
        -- sh -c 'uname -a && echo "Hello from Docker Bacalhau!"'
```


### Structure of the command

`ghcr.io/bacalhau-project/bacalhau:latest `: Name of the Bacalhau Docker image

`--id-only`: Output only the job id

`--wait`: Wait for the job to finish

`ubuntu:latest.` Ubuntu container

 `--`: Separate Bacalhau parameters from the command to be executed inside the container

 `sh -c 'uname -a && echo "Hello from Docker Bacalhau!"'`: The command executed inside the container

Let's have a look at the command execution in the terminal:

```shell
13:53:46.478 | INF pkg/repo/fs.go:81 > Initializing repo at '/root/.bacalhau' for environment 'production'
ab95a5cc-e6b7-40f1-957d-596b02251a66
```

The output you're seeing is in two parts:
**The first line:** `13:53:46.478 | INF pkg/repo/fs.go:81 > Initializing repo at '/root/.bacalhau' for environment 'production'` is an informational message indicating the initialization of a repository at the specified directory `('/root/.bacalhau')` for the `production` environment.
**The second line:** `ab95a5cc-e6b7-40f1-957d-596b02251a66` is a `job ID`, which represents the result of executing a command inside a Docker container. It can be used to obtain additional information about the executed job or to access the job's results. We store that in an environment variable so that we can reuse it later on (env: `JOB_ID=ab95a5cc-e6b7-40f1-957d-596b02251a66`)

To print out the **content of the Job ID**, run the following command:

```shell
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    describe ab95a5cc-e6b7-40f1-957d-596b02251a66 \
        | grep -A 2 "stdout: |"

Expected Output:
stdout: |
        Linux fff680719453 6.2.0-1019-gcp #21~22.04.1-Ubuntu SMP Thu Nov 16 18:18:34 UTC 2023 x86_64 x86_64 x86_64 GNU/Linux
        Hello from Docker Bacalhau!
```

## 4. Submit a Job With Output Files

One inconvenience that you'll see is that you'll need to mount directories into the container to access files. This is because the container is running in a separate environment from your host machine. Let's take a look at the example below:

The first part of the example should look familiar, except for the Docker commands.


```shell
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    docker run \
        --id-only \
        --wait \
        --gpu 1 \
        ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1 -- \
            python main.py --o ./outputs --p "A Docker whale and a cod having a conversation about the state of the ocean"
```

When a job is submitted, Bacalhau prints out the related `job_id` (`a46a9aa9-63ef-486a-a2f8-6457d7bafd2e`):

```shell
09:05:58.434 | INF pkg/repo/fs.go:81 > Initializing repo at '/root/.bacalhau' for environment 'production'
a46a9aa9-63ef-486a-a2f8-6457d7bafd2e
```


## 5. Checking the State of your Jobs

**Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    list $JOB_ID \
```

When it says `Completed`, that means the job is done, and we can get the results.

**Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    describe $JOB_ID \

```

**Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in the `result` directory.


```bash
%%bash
bacalhau get ${JOB_ID} --output-dir result
```

After the download has finished you should see the following contents in the results directory.





![png](index_files/index_25_0.png)




## Support

If have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).
