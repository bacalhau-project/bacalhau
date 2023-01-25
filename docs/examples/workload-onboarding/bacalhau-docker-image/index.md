---
sidebar_label: "Bacalhau Docker Image"
sidebar_position: 99
description: How to use the Bacalhau Docker image
---
# Bacalhau Docker Image

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/bacalhau-docker-image/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/bacalhau-docker-image/index.ipynb)

This example shows you how to run some common client-side Bacalhau tasks using the Bacalhau Docker image.

## Prerequisites

* [Install Docker](https://docs.docker.com/get-docker/)

## Pull the Container

The first step is to pull the Bacalhau Docker image from the [Github container registry](https://github.com/orgs/bacalhau-project/packages/container/package/bacalhau).


```bash
docker pull ghcr.io/bacalhau-project/bacalhau:latest
```

    latest: Pulling from bacalhau-project/bacalhau
    Digest: sha256:bdf27fb3af4accee119941eefa719d4d2892b6774f1be603a02e6da6bb56c492
    Status: Image is up to date for ghcr.io/bacalhau-project/bacalhau:latest
    ghcr.io/bacalhau-project/bacalhau:latest


You can also pull a specific version of the image, e.g.:

```bash
docker pull ghcr.io/bacalhau-project/bacalhau:v0.3.16
```

:::warning
Remember that the "latest" tag is just a string. It doesn't refer to the latest version of the Bacalhau client, it refers to an image that has the "latest" tag. Therefore, if your machine has already downloaded the "latest" image, it won't download it again. To force a download, you can use the `--no-cache` flag.
:::

## Running Common Bacalhau Commands

Now we're ready to run some common Bacalhau tasks.

### Check the Version

It's always a good idea to check the version of the client you're using. Mismatched versions can cause unexpected behaviour; a bit like trying to shove a US quarter into a UK pound coin slot. It works, but you're probably not getting your coin back.


```bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest version
```

    Client Version: v0.3.16
    Server Version: v0.3.16


### Submit a Docker Job

Yes, that's right, we can submit a Docker job using the Bacalhau Docker image...

In this example, I run an ubuntu-based job that echo's some stuff.


```bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    docker run \
        --id-only \
        --wait \
        ubuntu:latest -- \
            sh -c 'uname -a && echo "Hello from Docker Bacalhau!"'
```

    env: JOB_ID=a53f290c-1b45-4454-be6e-050c0f4e8741



```bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    describe $JOB_ID \
        | grep -A 2 "stdout: |"
```

                  stdout: |
                    Linux 8eac0284b095 5.15.0-1027-gcp #34-Ubuntu SMP Fri Jan 6 01:03:08 UTC 2023 x86_64 x86_64 x86_64 GNU/Linux
                    Hello from Docker Bacalhau!


### Sumbit a Job With Output Files

One inconvenience that you'll see is that you'll need to mount directories into the container to access files. This is because the container is running in a separate environment to your host machine. The example below steals one of the examples from the stable-diffusion demo.

The first part of the example should look familiar, except for the Docker commands.


```bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    docker run \
        --id-only \
        --wait \
        --gpu 1 \
        ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1 -- \
            python main.py --o ./outputs --p "A Docker whale and a cod having a conversation about the state of the ocean"
```

    env: JOB_ID=ca7cd1b8-9a84-4f1c-b180-a30785fb0990


This is where things get a bit more spicy. We need to mount a directory into the container so when you retrieve the results they are copied to the host machine.


```bash
docker run -t -v $(pwd)/results:/results ghcr.io/bacalhau-project/bacalhau:latest \
    get $JOB_ID --output-dir /results
```

    Fetching results of job 'ca7cd1b8-9a84-4f1c-b180-a30785fb0990'...
    2023/01/25 14:12:58 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.
    Results for job 'ca7cd1b8-9a84-4f1c-b180-a30785fb0990' have been written to...
    /results





    
![png](index_files/index_16_0.png)
    



I'm not entirely sure what's going on with that image. It looks like half an Orca on a beach holiday. But it's a good example of how to mount the results directory into the container. This pattern should work with any job that produces output files.

## Need Support?

If have questions or need support or guidance, please reach out to the [Bacalhau team via Slack (#bacalhau channel)](https://filecoin.io/slack)

