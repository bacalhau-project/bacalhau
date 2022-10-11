---
sidebar_label: "Python Hello World"
sidebar_position: 1
---
# Python Hello World

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/trivial-python/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/trivial-python/index.ipynb)

This example serves as an introduction to Bacalhau, running a Python file hosted on a website.


## Prerequisites

Make sure you have the latest `bacalhau` client installed by following the [getting started instructions](../../../getting-started/installation) or using the installation command below (which installs Bacalhau local to the notebook).


```python
!command -v bacalhau >/dev/null 2>&1 || (export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
path=!echo $PATH
%env PATH=./:{path[0]}
```

    env: PATH=./:./:./:/Users/phil/.pyenv/versions/3.9.7/bin:/opt/homebrew/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/bin:/Users/phil/.gvm/bin:/opt/homebrew/opt/findutils/libexec/gnubin:/opt/homebrew/opt/coreutils/libexec/gnubin:/opt/homebrew/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/bin:/Users/phil/.pyenv/shims:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/Library/TeX/texbin:/usr/local/MacGPG2/bin:/Users/phil/.nexustools


## Hello, world

For this example, we'll be using a very simple Python script which displays the [traditional first greeting](https://en.wikipedia.org/wiki/%22Hello,_World!%22_program).


```python
%cat hello-world.py
```

    print("Hello, world!")

## Submit the workload

To submit a workload to Bacalhau you can use the `bacalhau docker run` command. While you'll mainly be passing input data into the container using [content identifier (CID)](https://github.com/multiformats/cid) volumes, we will be using the `-u URL:path` argument for a simplicity. This results in Bacalhau mounting a *data volume* inside the container. By default, Bacalhau mounts the input volume at the path `/inputs` inside the container.

Note that [Bacalhau overwrites the default entrypoint](https://github.com/filecoin-project/bacalhau/blob/v0.2.3/cmd/bacalhau/docker_run.go#L64), so we must run the full command after the `--` argument.


```bash
bacalhau docker run \
  --input-urls https://raw.githubusercontent.com/bacalhau-project/examples/trivial-python-example/workload-onboarding/trivial-python/hello-world.py \
  python:3.10-slim -- python3 /inputs/hello-world.py
```

    Job successfully submitted. Job ID: b65c5d6f-9695-40a5-9c53-b91b306cbeea
    Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):
    
    	       Creating job for submission ... done ✅
    	       Finding node(s) for the job ... done ✅
    	             Node accepted the job ... done ✅
    	                                   ... done ✅
    	   Job finished, verifying results ... done ✅
    	      Results accepted, publishing ... done ✅
    	                                  
    Results CID: QmehTNF6ogbESt26EgrSw9YGrApneSWhPesqw1A5T6ezBe
    Job Results By Node:
    Node QmXaXu9N:
      Shard 0:
        Status: Cancelled
        No RunOutput for this shard
    Node QmYgxZiy:
      Shard 0:
        Status: Completed
        Container Exit Code: 0
        Stdout:
          Hello, world!
        Stderr: <NONE>
    Node QmdZQ7Zb:
      Shard 0:
        Status: Cancelled
        No RunOutput for this shard
    
    To download the results, execute:
      bacalhau get b65c5d6f-9695-40a5-9c53-b91b306cbeea
    
    To get more details about the run, execute:
      bacalhau describe b65c5d6f-9695-40a5-9c53-b91b306cbeea


## Get Results

If you look at the `stdout` from the previous command you'll see that it successfully ran the python file.
