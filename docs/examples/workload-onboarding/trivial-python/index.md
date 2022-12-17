---
sidebar_label: "Python - Hello World"
sidebar_position: 1
description: How to run a Python file hosted on Bacalhau
---
# Python Hello World

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/trivial-python/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/trivial-python/index.ipynb)

This example serves as an introduction to Bacalhau. Here, you'll be running a Python file hosted on a website on Bacalhau.

:::tip
You can run this code on your command line interface (CLI), or you can use the **[Google Colab](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/trivial-python/index.ipynb)** or **[Binder](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/trivial-python/index.ipynb)** notebooks provided at the top of this example to test the code.
:::

## Prerequisites

* [The Bacalhau client](https://docs.bacalhau.org/getting-started/installation)


```python
!command -v bacalhau >/dev/null 2>&1 || (export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
path=!echo $PATH
%env PATH=./:{path[0]}
```

    env: PATH=./:./:/home/gitpod/.pyenv/versions/3.8.13/bin:/home/gitpod/.pyenv/libexec:/home/gitpod/.pyenv/plugins/python-build/bin:/home/gitpod/.pyenv/plugins/pyenv-virtualenv/bin:/home/gitpod/.pyenv/plugins/pyenv-update/bin:/home/gitpod/.pyenv/plugins/pyenv-installer/bin:/home/gitpod/.pyenv/plugins/pyenv-doctor/bin:/home/gitpod/.pyenv/shims:/ide/bin/remote-cli:/home/gitpod/.nix-profile/bin:/home/gitpod/.local/bin:/home/gitpod/.sdkman/candidates/maven/current/bin:/home/gitpod/.sdkman/candidates/java/current/bin:/home/gitpod/.sdkman/candidates/gradle/current/bin:/workspace/.cargo/bin:/home/gitpod/.rvm/gems/ruby-3.1.2/bin:/home/gitpod/.rvm/gems/ruby-3.1.2@global/bin:/home/gitpod/.rvm/rubies/ruby-3.1.2/bin:/home/gitpod/.pyenv/plugins/pyenv-virtualenv/shims:/home/gitpod/.pyenv/shims:/workspace/go/bin:/home/gitpod/.nix-profile/bin:/ide/bin/remote-cli:/home/gitpod/go/bin:/home/gitpod/go-packages/bin:/home/gitpod/.nvm/versions/node/v16.18.1/bin:/home/gitpod/.yarn/bin:/home/gitpod/.pnpm:/home/gitpod/.pyenv/bin:/workspace/.rvm/bin:/home/gitpod/.cargo/bin:/home/linuxbrew/.linuxbrew/bin:/home/linuxbrew/.linuxbrew/sbin/:/home/gitpod/.local/bin:/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/home/gitpod/.nvm/versions/node/v16.18.1/bin:/home/gitpod/.rvm/bin


## Hello, world

For this example, we'll be using a very simple Python script which displays the [traditional first greeting](https://en.wikipedia.org/wiki/%22Hello,_World!%22_program).


```python
%cat hello-world.py
```

    print("Hello, world!")

## Submit the workload

To submit a workload to Bacalhau you can use the `bacalhau docker run` command. While you'll mainly be passing input data into the container using [content identifier (CID)](https://github.com/multiformats/cid) volumes, we will be using the `-u URL:path` [argument](https://docs.bacalhau.org/all-flags#docker-run) for simplicity. This results in Bacalhau mounting a *data volume* inside the container. By default, Bacalhau mounts the input volume at the path `/inputs` inside the container.

:::info
[Bacalhau overwrites the default entrypoint](https://github.com/filecoin-project/bacalhau/blob/v0.2.3/cmd/bacalhau/docker_run.go#L64), so we must run the full command after the `--` argument.
:::


```bash
bacalhau docker run \
  --id-only \
  --input-urls https://raw.githubusercontent.com/bacalhau-project/examples/151eebe895151edd83468e3d8b546612bf96cd05/workload-onboarding/trivial-python/hello-world.py \
  python:3.10-slim -- python3 /inputs/hello-world.py
```

## Get Results

After the job has finished processing, the next step is to use the `get` verb to download your outputs locally. 

You can run the `bacalhau get` directly as shown below


```python
%env JOB_ID={job_id}
```

    env: JOB_ID=bde362b4-cc5a-4c3a-8042-122cc3c0c2ea



```bash
bacalhau describe ${JOB_ID}
```

    APIVersion: V1beta1
    Metadata:
      ClientID: 81568beeb7c8626d4565627ca0fd4b76fd18fec20a37c3b6a9b02bc03cbce5ae
      CreatedAt: "2022-12-14T11:01:02.501686348Z"
      ID: bde362b4-cc5a-4c3a-8042-122cc3c0c2ea
    Spec:
      Deal:
        Concurrency: 1
      Docker:
        Entrypoint:
        - python3
        - /inputs/hello-world.py
        Image: python:3.10-slim
      Engine: Docker
      ExecutionPlan:
        ShardsTotal: 1
      Language:
        JobContext: {}
      Publisher: Estuary
      Resources:
        GPU: ""
      Sharding:
        BatchSize: 1
        GlobPatternBasePath: /inputs
      Timeout: 1800
      Verifier: Noop
      Wasm: {}
      inputs:
      - StorageSource: URLDownload
        URL: https://raw.githubusercontent.com/bacalhau-project/examples/151eebe895151edd83468e3d8b546612bf96cd05/workload-onboarding/trivial-python/hello-world.py
        path: /inputs
      outputs:
      - Name: outputs
        StorageSource: IPFS
        path: /outputs
    Status:
      JobState:
        Nodes:
          QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG:
            Shards:
              "0":
                NodeId: QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG
                PublishedResults:
                  CID: QmehTNF6ogbESt26EgrSw9YGrApneSWhPesqw1A5T6ezBe
                  Name: job-bde362b4-cc5a-4c3a-8042-122cc3c0c2ea-shard-0-host-QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG
                  StorageSource: IPFS
                RunOutput:
                  exitCode: 0
                  runnerError: ""
                  stderr: ""
                  stderrtruncated: false
                  stdout: |
                    Hello, world!
                  stdouttruncated: false
                State: Completed
                VerificationResult:
                  Complete: true
                  Result: true
          QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF:
            Shards:
              "0":
                NodeId: QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF
                PublishedResults: {}
                State: Cancelled
                VerificationResult: {}
          QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3:
            Shards:
              "0":
                NodeId: QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
                PublishedResults: {}
                State: Cancelled
                VerificationResult: {}
          QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL:
            Shards:
              "0":
                NodeId: QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
                PublishedResults: {}
                State: Cancelled
                VerificationResult: {}
      Requester:
        RequesterNodeID: QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
        RequesterPublicKey: CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDF5pYaTdt4UwzslPRDr8XFXv1clttGGIWENlnECLVqswrQVp5xrMsI/57MrJUsMADsz6a/cip9LOxiym3bZVIeZ5WmbrFp48F0Wb7RCELAsUcl/zx4FHCK+A2FKqmuhpY9NyVNGupIDBUCPvCWsDb87Ur//a9BdCOETuab4577e3vFCa3fE+9jn5Fuhoa0m5Z7GfuZtU0W2FX6nc4VIfseMWrWpHD+Bqe/kXs+8RFBVW2AYyzO8lCcHtRC4Lb1Ml1V5hcoAN1pe5yfVuPlT9qyAgCnH3nIIWtvEYz8BnSDgXXTHdT+N+6lrm9oMglNh7TpT6ZbmpioIbJalelAyhG3AgMBAAE=


Alternatively, you can create a directory that will store our job outputs.


```bash
rm -rf results && mkdir results
bacalhau get ${JOB_ID} --output-dir results
```

    Fetching results of job 'bde362b4-cc5a-4c3a-8042-122cc3c0c2ea'...
    Results for job 'bde362b4-cc5a-4c3a-8042-122cc3c0c2ea' have been written to...
    results


At this point, the outputs will be downloaded locally. Each job creates 3 sub_folders: the *combined_results*, *per_shard* files, and the *raw* directory. In each of this sub_folders, you'll find the *stdout* and *stderr*

For the scope this of this guide, we will only look at the **stdout** file. You can go directly to the file folder to inspect the content of the file or use the code belolow


```bash

cat results/combined_results/stdout

```

    Hello, world!


## Need Support?

If have questions or need support or guidance, please reach out to the [Bacalhau team via Slack (#bacalhau channel)](https://filecoin.io/slack)

