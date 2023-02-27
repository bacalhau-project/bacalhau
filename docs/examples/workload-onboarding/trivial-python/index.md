---
sidebar_label: "Python - Hello World"
sidebar_position: 1
description: How to run a Python file hosted on Bacalhau
---
# Python Hello World

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/trivial-python/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/trivial-python/index.ipynb)

This example tutorial serves as an introduction to Bacalhau. Here, you'll be running a Python file hosted on a website on Bacalhau.



## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Creating a Hello World File

We'll be using a very simple Python script which displays the [traditional first greeting](https://en.wikipedia.org/wiki/%22Hello,_World!%22_program).


```python
%cat hello-world.py
```

    print("Hello, world!")

## Submit the workload

To submit a workload to Bacalhau you can use the `bacalhau docker run` command. 


```bash
%%bash --out job_id
bacalhau docker run \
  --id-only \
  --input-urls https://raw.githubusercontent.com/bacalhau-project/examples/151eebe895151edd83468e3d8b546612bf96cd05/workload-onboarding/trivial-python/hello-world.py \
  python:3.10-slim -- python3 /inputs/hello-world.py
```

When a job is sumbitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%env JOB_ID={job_id}
```

    env: JOB_ID=c2f245d6-43a6-43ec-9a3b-7ce9b6242c88


The `bacalhau docker run` command allows to pass input data into the container using [content identifier (CID)](https://github.com/multiformats/cid) volumes, we will be using the `-u URL:path` [argument](https://docs.bacalhau.org/all-flags#docker-run) for simplicity. This results in Bacalhau mounting a *data volume* inside the container. By default, Bacalhau mounts the input volume at the path `/inputs` inside the container.

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

    APIVersion: V1beta1
    Metadata:
      ClientID: 77cf46c04f88ffb1c3e0e4b6e443724e8d2d87074d088ef1a6294a448fa85d2e
      CreatedAt: "2023-01-20T13:24:59.165644684Z"
      ID: c2f245d6-43a6-43ec-9a3b-7ce9b6242c88
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
          QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT:
            Shards:
              "0":
                NodeId: QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT
                PublishedResults: {}
                State: Cancelled
                VerificationResult: {}
          QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG:
            Shards:
              "0":
                NodeId: QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG
                PublishedResults: {}
                State: Cancelled
                VerificationResult: {}
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
                PublishedResults:
                  CID: QmehTNF6ogbESt26EgrSw9YGrApneSWhPesqw1A5T6ezBe
                  Name: job-c2f245d6-43a6-43ec-9a3b-7ce9b6242c88-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
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
      Requester:
        RequesterNodeID: QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
        RequesterPublicKey: CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDVRKPgCfY2fgfrkHkFjeWcqno+MDpmp8DgVaY672BqJl/dZFNU9lBg2P8Znh8OTtHPPBUBk566vU3KchjW7m3uK4OudXrYEfSfEPnCGmL6GuLiZjLf+eXGEez7qPaoYqo06gD8ROdD8VVse27E96LlrpD1xKshHhqQTxKoq1y6Rx4DpbkSt966BumovWJ70w+Nt9ZkPPydRCxVnyWS1khECFQxp5Ep3NbbKtxHNX5HeULzXN5q0EQO39UN6iBhiI34eZkH7PoAm3Vk5xns//FjTAvQw6wZUu8LwvZTaihs+upx2zZysq6CEBKoeNZqed9+Tf+qHow0P5pxmiu+or+DAgMBAAE=


- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir results
bacalhau get ${JOB_ID} --output-dir results
```

    Fetching results of job 'c2f245d6-43a6-43ec-9a3b-7ce9b6242c88'...
    Results for job 'c2f245d6-43a6-43ec-9a3b-7ce9b6242c88' have been written to...
    results


    2023/01/20 13:25:06 CleanupManager.fnsMutex violation CRITICAL section took 43.424ms 43424000 (threshold 10ms)


## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**,**per_shard files**, and the **raw** directory. To view the file, run the following command:


```bash

%%bash
cat results/combined_results/stdout

```

    Hello, world!


## Need Support?

For questions, feedback, please reach out in our [forum](https://github.com/filecoin-project/bacalhau/discussions)
