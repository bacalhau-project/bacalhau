---
sidebar_label: "Python File"
sidebar_position: 12
description: How to run a Python file hosted on Bacalhau
---
# Running a Python Script


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

This tutorial serves as an introduction to Bacalhau. In this example, you'll be executing a simple  "Hello, World!" Python script hosted on a website on Bacalhau.

### Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)

## 1. Running Python Locally

We'll be using a very simple Python script that displays the [traditional first greeting](https://en.wikipedia.org/wiki/%22Hello,_World!%22_program). Create a file called `hello-world.py`:

```python
%%writefile hello-world.py
print("Hello, world!")
```

```python
%cat hello-world.py
```

Running the script to print out the output:

```bash
%%bash
python3 hello-world.py
```
After the script has run successfully locally we can now run it on Bacalhau.

## 2. Running a Bacalhau Job



To submit a workload to Bacalhau you can use the `bacalhau docker run` command. This command allows passing input data into the container using [content identifier (CID)](https://github.com/multiformats/cid) volumes, we will be using the `--input URL:path` [argument](../../../dev/cli-reference/all-flags.md#docker-run) for simplicity. This results in Bacalhau mounting a *data volume* inside the container. By default, Bacalhau mounts the input volume at the path `/inputs` inside the container.

:::info
[Bacalhau overwrites the default entrypoint](https://github.com/filecoin-project/bacalhau/blob/v0.2.3/cmd/bacalhau/docker_run.go#L64), so we must run the full command after the `--` argument.
:::


```bash
%%bash --out job_id
bacalhau docker run \
    --id-only \
    --input https://raw.githubusercontent.com/bacalhau-project/examples/151eebe895151edd83468e3d8b546612bf96cd05/workload-onboarding/trivial-python/hello-world.py \
    python:3.10-slim \
    -- python3 /inputs/hello-world.py
```

### Structure of the command

`bacalhau docker run`: call to Bacalhau

`--id-only`: specifies that only the job identifier (job_id) will be returned after executing the container, not the entire output

`--input https://raw.githubusercontent.com/bacalhau-project/examples/151eebe895151edd83468e3d8b546612bf96cd05/workload-onboarding/trivial-python/hello-world.py \`: indicates where to get the input data for the container. In this case, the input data is downloaded from the specified URL, which represents the Python script "hello-world.py".

`python:3.10-slim`: the Docker image that will be used to run the container. In this case, it uses the Python 3.10 image with a minimal set of components (slim).

`--`: This double dash is used to separate the Bacalhau command options from the command that will be executed inside the Docker container.

`python3 /inputs/hello-world.py`: running the `hello-world.py` Python script stored in `/inputs`.


When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on:


```python
%env JOB_ID={job_id}
```

## 3. Checking the State of your Jobs

**Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --no-style
```

When it says `Published` or `Completed`, that means the job is done, and we can get the results.

**Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe ${JOB_ID}
```

**Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory (`results`) and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir results
bacalhau get ${JOB_ID} --output-dir results
```

## 4. Viewing your Job Output

To view the file, run the following command:


```bash
%%bash
cat results/stdout
```

## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).
