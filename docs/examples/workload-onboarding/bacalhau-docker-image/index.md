---
sidebar_label: "Bacalhau Docker Image"
sidebar_position: 1
description: How to use the Bacalhau Docker image
---
# Bacalhau Docker Image

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/bacalhau-docker-image/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/bacalhau-docker-image/index.ipynb)
[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

This example shows you how to run some common client-side Bacalhau tasks using the Bacalhau Docker image.

## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Pull the Docker image

The first step is to pull the Bacalhau Docker image from the [Github container registry](https://github.com/orgs/bacalhau-project/packages/container/package/bacalhau).


```bash
%%bash
docker pull ghcr.io/bacalhau-project/bacalhau:latest
```

You can also pull a specific version of the image, e.g.:

```bash
docker pull ghcr.io/bacalhau-project/bacalhau:v0.3.16
```

:::warning
Remember that the "latest" tag is just a string. It doesn't refer to the latest version of the Bacalhau client, it refers to an image that has the "latest" tag. Therefore, if your machine has already downloaded the "latest" image, it won't download it again. To force a download, you can use the `--no-cache` flag.
:::

## Check version

Check the version of the Bacalhau client you are using.



```bash
%%bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest version
```

## Running a Bacalhau Job

To submit a bi to Bacalhau, we use the `bacalhau docker run` command. 


```bash
%%bash --out job_id
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    docker run \
        --id-only \
        --wait \
        ubuntu:latest -- \
            sh -c 'uname -a && echo "Hello from Docker Bacalhau!"'
```

In this example, I run an ubuntu-based job that echo's some stuff.

### Structure of the command

-  `--id-only......`: Output only the job id

- `ubuntu:latest.` Ubuntu container

- `ghcr.io/bacalhau-project/bacalhau:latest `: Name of the Bacalhau Docker image

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

To print out the content of the Job ID, run the following command:


```bash
%%bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    describe $JOB_ID \
        | grep -A 2 "stdout: |"
```

                  stdout: |
                    Linux 8eac0284b095 5.15.0-1027-gcp #34-Ubuntu SMP Fri Jan 6 01:03:08 UTC 2023 x86_64 x86_64 x86_64 GNU/Linux
                    Hello from Docker Bacalhau!


## Sumbit a Job With Output Files

One inconvenience that you'll see is that you'll need to mount directories into the container to access files. This is because the container is running in a separate environment to your host machine. Let's take a look at the example below:

The first part of the example should look familiar, except for the Docker commands.


```bash
%%bash --out job_id
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    docker run \
        --id-only \
        --wait \
        --gpu 1 \
        ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1 -- \
            python main.py --o ./outputs --p "A Docker whale and a cod having a conversation about the state of the ocean"
```


When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`. 


```bash
%%bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    list $JOB_ID \
        | grep -A 2 "stdout: |"
```

When it says `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    describe $JOB_ID \
        | grep -A 2 "stdout: |"
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
docker run -t -v $(pwd)/results:/results ghcr.io/bacalhau-project/bacalhau:latest \
    get $JOB_ID --output-dir /results
```

After the download has finished you should see the following contents in results directory. 




    
![png](index_files/index_24_0.png)
    



## Need Support?

If have questions or need support or guidance, please reach out to the [Bacalhau team via Slack (#bacalhau channel)](https://filecoin.io/slack)

