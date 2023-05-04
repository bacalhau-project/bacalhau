---
sidebar_label: "Python File"
sidebar_position: 12
description: How to run a Python file hosted on Bacalhau
---
# Running a Python Script


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

This example tutorial serves as an introduction to Bacalhau. Here, you'll be running a Python file hosted on a website on Bacalhau.

## TD;LR
A quick guide on how to run a hello world script on Bacalhau

## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Creating a Hello World File

We'll be using a very simple Python script which displays the [traditional first greeting](https://en.wikipedia.org/wiki/%22Hello,_World!%22_program).


```python
%cat hello-world.py
```

## Submit the workload

To submit a workload to Bacalhau you can use the `bacalhau docker run` command. 


```bash
%%bash --out job_id
bacalhau docker run \
  --id-only \
  --input https://raw.githubusercontent.com/bacalhau-project/examples/151eebe895151edd83468e3d8b546612bf96cd05/workload-onboarding/trivial-python/hello-world.py \
  python:3.10-slim -- python3 /inputs/hello-world.py
```

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%env JOB_ID={job_id}
```

The `bacalhau docker run` command allows to pass input data into the container using [content identifier (CID)](https://github.com/multiformats/cid) volumes, we will be using the `-i URL:path` [argument](https://docs.bacalhau.org/all-flags#docker-run) for simplicity. This results in Bacalhau mounting a *data volume* inside the container. By default, Bacalhau mounts the input volume at the path `/inputs` inside the container.

:::info
[Bacalhau overwrites the default entrypoint](https://github.com/filecoin-project/bacalhau/blob/v0.2.3/cmd/bacalhau/docker_run.go#L64), so we must run the full command after the `--` argument.
:::

## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`. 


```bash
%%bash
bacalhau list --id-filter=${JOB_ID} --no-style
```

When it says `Published` or `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe ${JOB_ID}
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir results
bacalhau get ${JOB_ID} --output-dir results
```

## Viewing your Job Output

To view the file, run the following command:


```bash

%%bash
cat results/stdout

```

## Need Support?

For questions, feedback, please reach out in our [forum](https://github.com/filecoin-project/bacalhau/discussions)
