---
sidebar_label: "Python - Hello World"
sidebar_position: 1
---
# Python Hello World

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/trivial-python/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/trivial-python/index.ipynb)

This example serves as an introduction to Bacalhau, running a Python file hosted on a website.


## Prerequisites

Make sure you have the latest `bacalhau` client installed by following the [getting started instructions](../../../getting-started/installation) or using the installation command below (which installs Bacalhau local to the notebook).

## Hello, world

For this example, we'll be using a very simple Python script which displays the [traditional first greeting](https://en.wikipedia.org/wiki/%22Hello,_World!%22_program).

    print("Hello, world!")

## Submit the workload

To submit a workload to Bacalhau you can use the `bacalhau docker run` command. While you'll mainly be passing input data into the container using [content identifier (CID)](https://github.com/multiformats/cid) volumes, we will be using the `-u URL:path` argument for a simplicity. This results in Bacalhau mounting a *data volume* inside the container. By default, Bacalhau mounts the input volume at the path `/inputs` inside the container.

Note that [Bacalhau overwrites the default entrypoint](https://github.com/filecoin-project/bacalhau/blob/v0.2.3/cmd/bacalhau/docker_run.go#L64), so we must run the full command after the `--` argument.


```bash
bacalhau docker run \
  --input-urls https://raw.githubusercontent.com/bacalhau-project/examples/151eebe895151edd83468e3d8b546612bf96cd05/workload-onboarding/trivial-python/hello-world.py \
  python:3.10-slim -- python3 /inputs/hello-world.py
```

    Job successfully submitted. Job ID: 15f57de4-4ea1-45ca-899d-fba08fb53420
    Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):
    
    	       Creating job for submission ... done ✅
    	       Finding node(s) for the job ... done ✅
    	             Node accepted the job ... done ✅
    	   Job finished, verifying results ... done ✅
    	      Results accepted, publishing ... done ✅
    	                                  
    Results CID: QmehTNF6ogbESt26EgrSw9YGrApneSWhPesqw1A5T6ezBe
    Job Results By Node:
    Node QmXaXu9N:
      Shard 0:
        Status: Completed
        Container Exit Code: 0
        Stdout:
          Hello, world!
        Stderr: <NONE>
    
    To download the results, execute:
      bacalhau get 15f57de4-4ea1-45ca-899d-fba08fb53420
    
    To get more details about the run, execute:
      bacalhau describe 15f57de4-4ea1-45ca-899d-fba08fb53420


## Get Results

If you look at the `stdout` from the previous command you'll see that it successfully ran the python file.
