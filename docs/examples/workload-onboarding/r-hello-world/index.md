---
sidebar_label: "R - Hello World"
sidebar_position: 50
---
# Running a Simple R Script in Bacalhau

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/r-hello-world/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/r-hello-world/index.ipynb)

You can use official Docker containers for each language like R or Python. In this example, we will use the official R container and run it on bacalhau. 

:::tip
Please make sure that you have all the dependencies installed and the scripts shouldn’t make any external requests because networking is disabled in Bacalhau.
:::

## Prerequisites

* A working R environment
* [Bacalhau](https://docs.bacalhau.org/getting-started/installation)

## 1. Running an R Script Locally

To install R follow these instructions [A Installing R and RStudio | Hands-On Programming with R](https://rstudio-education.github.io/hopr/starting.html). After R and RStudio is installed, create and run a script called hello.R.


```python
%%writefile hello.R
print("hello world")
```

    Overwriting hello.R


Run the script:


```bash
%%bash
Rscript hello.R
```

    [1] "hello world"


Recall that Bacalhau does now provide any external connectivity whilst running a job. So you must place the script in a container or, as shown below, upload the script to IPFS for long term storage. 

We've already uploaded the script to IPFS and the CID is: `QmVHSWhAL7fNkRiHfoEJGeMYjaYZUsKHvix7L54SptR8ie`. You can look at this by browsing to one of the HTTP IPFS proxies like [ipfs.io](https://cloudflare-ipfs.com/ipfs/QmVHSWhAL7fNkRiHfoEJGeMYjaYZUsKHvix7L54SptR8ie/) or [w3s.link](https://w3s.link/ipfs/QmVHSWhAL7fNkRiHfoEJGeMYjaYZUsKHvix7L54SptR8ie).

## 2. Running an R Script on Bacalhau**

Now it's time to run the script on the Bacalhau network. Bacalhau is a decentralised network of compute nodes. To run a job on Bacalhau you need to submit a job request.


```bash
%%bash --out job_id
bacalhau docker run \
--wait \
--id-only \
-v QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk:/hello.R \
r-base \
-- Rscript hello.R
```


```python
%env JOB_ID={job_id}
```

    env: JOB_ID=d6ad3239-31d7-4b44-8125-980e89b2dbbb



Running the commands will output a UUID (like `54506541-4eb9-45f4-a0b1-ea0aecd34b3e`). This is the ID of the job that was created. You can check the status of the job with the following command:



```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 14:26:05 [0m[97;40m d6ad3239 [0m[97;40m Docker r-base Rscrip... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmQ1Yci2Gbptoc... [0m



Where it says `Published`, that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```bash
%%bash
bacalhau describe  ${JOB_ID}
```

    APIVersion: V1beta1
    ClientID: 77cf46c04f88ffb1c3e0e4b6e443724e8d2d87074d088ef1a6294a448fa85d2e
    CreatedAt: "2022-11-23T14:26:05.742836012Z"
    Deal:
      Concurrency: 1
    ExecutionPlan:
      ShardsTotal: 1
    ID: d6ad3239-31d7-4b44-8125-980e89b2dbbb
    JobState:
      Nodes:
        QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG:
          Shards:
            "0":
              NodeId: QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG
              PublishedResults:
                CID: QmQ1Yci2GbptoccHy46txBK28gtnbKcb7nvFNHYpS6Gbn7
                Name: job-d6ad3239-31d7-4b44-8125-980e89b2dbbb-shard-0-host-QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG
                StorageSource: IPFS
              RunOutput:
                exitCode: 0
                runnerError: ""
                stderr: ""
                stderrtruncated: false
                stdout: |
                  [1] "hello world"
                stdouttruncated: false
              State: Completed
              Status: 'Got results proposal of length: 0'
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
    RequesterNodeID: QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
    RequesterPublicKey: CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDF5pYaTdt4UwzslPRDr8XFXv1clttGGIWENlnECLVqswrQVp5xrMsI/57MrJUsMADsz6a/cip9LOxiym3bZVIeZ5WmbrFp48F0Wb7RCELAsUcl/zx4FHCK+A2FKqmuhpY9NyVNGupIDBUCPvCWsDb87Ur//a9BdCOETuab4577e3vFCa3fE+9jn5Fuhoa0m5Z7GfuZtU0W2FX6nc4VIfseMWrWpHD+Bqe/kXs+8RFBVW2AYyzO8lCcHtRC4Lb1Ml1V5hcoAN1pe5yfVuPlT9qyAgCnH3nIIWtvEYz8BnSDgXXTHdT+N+6lrm9oMglNh7TpT6ZbmpioIbJalelAyhG3AgMBAAE=
    Spec:
      Docker:
        Entrypoint:
        - Rscript
        - hello.R
        Image: r-base
      Engine: Docker
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
      - CID: QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk
        StorageSource: IPFS
        path: /hello.R
      outputs:
      - Name: outputs
        StorageSource: IPFS
        path: /outputs


Since there is no error we can’t see any error instead we see the state of our job to be complete, that means 
we can download the results!
we create a temporary directory to save our results


```bash
%%bash
rm -rf results && mkdir results
```

To Download the results of your job, run the following command:


```bash
%%bash
bacalhau get ${JOB_ID} --output-dir results
```

    Fetching results of job 'd6ad3239-31d7-4b44-8125-980e89b2dbbb'...
    Results for job 'd6ad3239-31d7-4b44-8125-980e89b2dbbb' have been written to...
    results


After the download has finished you should 
see the following contents in results directory


```bash
%%bash
ls results/
```

    combined_results
    per_shard
    raw


Viewing the result


```bash
%%bash
cat results/combined_results/stdout
```

    [1] "hello world"


### Bonus: Futureproofing your R Scripts

You can generate the the job request with the following command. This will allow you to re-run that job in the future.


```bash
%%bash
bacalhau describe ${JOB_ID} --spec > job.yaml
```


```bash
%%bash
cat job.yaml
```

    APIVersion: V1beta1
    ClientID: 77cf46c04f88ffb1c3e0e4b6e443724e8d2d87074d088ef1a6294a448fa85d2e
    CreatedAt: "2022-11-23T14:26:05.742836012Z"
    Deal:
      Concurrency: 1
    ExecutionPlan:
      ShardsTotal: 1
    ID: d6ad3239-31d7-4b44-8125-980e89b2dbbb
    JobState:
      Nodes:
        QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG:
          Shards:
            "0":
              NodeId: QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG
              PublishedResults:
                CID: QmQ1Yci2GbptoccHy46txBK28gtnbKcb7nvFNHYpS6Gbn7
                Name: job-d6ad3239-31d7-4b44-8125-980e89b2dbbb-shard-0-host-QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG
                StorageSource: IPFS
              RunOutput:
                exitCode: 0
                runnerError: ""
                stderr: ""
                stderrtruncated: false
                stdout: |
                  [1] "hello world"
                stdouttruncated: false
              State: Completed
              Status: 'Got results proposal of length: 0'
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
    RequesterNodeID: QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
    RequesterPublicKey: CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDF5pYaTdt4UwzslPRDr8XFXv1clttGGIWENlnECLVqswrQVp5xrMsI/57MrJUsMADsz6a/cip9LOxiym3bZVIeZ5WmbrFp48F0Wb7RCELAsUcl/zx4FHCK+A2FKqmuhpY9NyVNGupIDBUCPvCWsDb87Ur//a9BdCOETuab4577e3vFCa3fE+9jn5Fuhoa0m5Z7GfuZtU0W2FX6nc4VIfseMWrWpHD+Bqe/kXs+8RFBVW2AYyzO8lCcHtRC4Lb1Ml1V5hcoAN1pe5yfVuPlT9qyAgCnH3nIIWtvEYz8BnSDgXXTHdT+N+6lrm9oMglNh7TpT6ZbmpioIbJalelAyhG3AgMBAAE=
    Spec:
      Docker:
        Entrypoint:
        - Rscript
        - hello.R
        Image: r-base
      Engine: Docker
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
      - CID: QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk
        StorageSource: IPFS
        path: /hello.R
      outputs:
      - Name: outputs
        StorageSource: IPFS
        path: /outputs

