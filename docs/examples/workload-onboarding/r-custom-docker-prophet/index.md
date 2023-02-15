---
sidebar_label: "R - Facebook Prophet - Custom Container"
sidebar_position: 51
---
# Building and Running your Custom R Containers on Bacalhau

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/r-custom-docker-prophet/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/r-custom-docker-prophet/index.ipynb)

## Introduction

This example will walk you through building Time Series Forecasting using [Prophet](https://github.com/facebook/prophet).

Prophet is a forecasting procedure implemented in R and Python. It is fast and provides completely automated forecasts that can be tuned by hand by data scientists and analysts.

### TL;DR

```bash
bacalhau docker run -v QmY8BAftd48wWRYDf5XnZGkhwqgjpzjyUG3hN1se6SYaFt:/example_wp_log_R.csv ghcr.io/bacalhau-project/examples/r-prophet:0.0.2 -- Rscript Saturating-Forecasts.R "/example_wp_log_R.csv" "/outputs/output0.pdf" "/outputs/output1.pdf"
```

## Prerequisites

* A working R environment
* [Docker](https://docs.docker.com/get-docker/)
* [Bacalhau](https://docs.bacalhau.org/getting-started/installation)

## 1. Running Prophet in R Locally

Open R studio or R supported IDE. If you want to run this on a notebook server, then make sure you use an R kernel.

Prophet is a CRAN package so you can use install.packages to install the prophet package.


```bash
%%bash
R -e "install.packages('prophet',dependencies=TRUE, repos='http://cran.rstudio.com/')"
```


After installation is finished, you can download the example data that is stored in IPFS.


```bash
%%bash
wget https://w3s.link/ipfs/QmZiwZz7fXAvQANKYnt7ya838VPpj4agJt5EDvRYp3Deeo/example_wp_log_R.csv
```

The code below instantiates the library and fits a model to the data.


```bash
%%bash
mkdir -p outputs
mkdir -p R
```


```python
%%writefile Saturating-Forecasts.R
library('prophet')

args = commandArgs(trailingOnly=TRUE)
args

input = args[1]
output = args[2]
output1 = args[3]


I <- paste("", input, sep ="")

O <- paste("", output, sep ="")

O1 <- paste("", output1 ,sep ="")


df <- read.csv(I)

df$cap <- 8.5
m <- prophet(df, growth = 'logistic')

future <- make_future_dataframe(m, periods = 1826)
future$cap <- 8.5
fcst <- predict(m, future)
pdf(O)
plot(m, fcst)
dev.off()

df$y <- 10 - df$y
df$cap <- 6
df$floor <- 1.5
future$cap <- 6
future$floor <- 1.5
m <- prophet(df, growth = 'logistic')
fcst <- predict(m, future)
pdf(O1)
plot(m, fcst)
dev.off()
```

    Writing Saturating-Forecasts.R



```bash
%%bash
Rscript Saturating-Forecasts.R "example_wp_log_R.csv" "outputs/output0.pdf" "outputs/output1.pdf"
```

## 2. Running R Prophet on Bacalhau

To use Bacalhau, you need to package your code in an appropriate format. The developers have already pushed a container for you to use, but if you want to build your own, you can follow the steps below. You can view a [dedicated container example](../custom-containers/index.md) in the documentation.

### Dockerfile

In this step, you will create a `Dockerfile` to create an image. The `Dockerfile` is a text document that contains the commands used to assemble the image. First, create the `Dockerfile`.

```
FROM r-base
RUN R -e "install.packages('prophet',dependencies=TRUE, repos='http://cran.rstudio.com/')"
RUN mkdir /R
RUN mkdir /outputs
COPY Saturating-Forecasts.R R
WORKDIR /R
```

Next, add your desired configuration to the `Dockerfile`. These commands specify how the image will be built, and what extra requirements will be included. We use r-base as the base image, and then install the prophet package. We then copy the R script into the container and set the working directory to the R folder.

We've already pushed this image to GHCR, but for posterity, you'd use a command like this to update it:

```bash
docker buildx build --platform linux/amd64 --push -t ghcr.io/bacalhau-project/examples/r-prophet:0.0.1 .
```

After you have built the container successfully, the next step is to test it locally and then push it docker hub

### Fitting a Prophet Model on Bacalhau

The following command passes a prompt to the model and generates the results in the outputs directory. It takes approximately 2 minutes to run.


```bash
%%bash --out job_id
bacalhau docker run \
    --wait \
    --id-only \
    -v QmY8BAftd48wWRYDf5XnZGkhwqgjpzjyUG3hN1se6SYaFt:/example_wp_log_R.csv \
    ghcr.io/bacalhau-project/examples/r-prophet:0.0.2 \
    -- Rscript Saturating-Forecasts.R "/example_wp_log_R.csv" "/outputs/output0.pdf" "/outputs/output1.pdf"
```

Running the commands will output a UUID that represents the job that was created. You can check the status of the job with the following command:


```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 15:10:22 [0m[97;40m 0316d0c2 [0m[97;40m Docker jsace/r-proph... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmYwR3uaSnhLpE... [0m



Where it says `Completed`, that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```bash
%%bash
bacalhau describe ${JOB_ID}
```

    APIVersion: V1alpha1
    ClientID: 77cf46c04f88ffb1c3e0e4b6e443724e8d2d87074d088ef1a6294a448fa85d2e
    CreatedAt: "2022-11-11T15:10:22.177011613Z"
    Deal:
      Concurrency: 1
    ExecutionPlan:
      ShardsTotal: 1
    ID: 0316d0c2-162d-4c57-9c10-391c908f981d
    JobState:
      Nodes:
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
                CID: QmYwR3uaSnhLpEZYDdUGXQMVCuCmsd8Rc4LHsuHL6pSUz3
                Name: job-0316d0c2-162d-4c57-9c10-391c908f981d-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
                StorageSource: IPFS
              RunOutput:
                exitCode: 0
                runnerError: ""
                stderr: |-
                  Loading required package: Rcpp
                  Loading required package: rlang
                  Disabling daily seasonality. Run prophet with daily.seasonality=TRUE to override this.
                  Disabling daily seasonality. Run prophet with daily.seasonality=TRUE to override this.
                stderrtruncated: false
                stdout: "[1] \"example_wp_log_R.csv\" \"outputs/output0.pdf\"  \"outputs/output1.pdf\"
                  \nnull device \n          1 \nnull device \n          1"
                stdouttruncated: false
              State: Completed
              Status: 'Got results proposal of length: 0'
              VerificationResult:
                Complete: true
                Result: true
    RequesterNodeID: QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
    RequesterPublicKey: CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDVRKPgCfY2fgfrkHkFjeWcqno+MDpmp8DgVaY672BqJl/dZFNU9lBg2P8Znh8OTtHPPBUBk566vU3KchjW7m3uK4OudXrYEfSfEPnCGmL6GuLiZjLf+eXGEez7qPaoYqo06gD8ROdD8VVse27E96LlrpD1xKshHhqQTxKoq1y6Rx4DpbkSt966BumovWJ70w+Nt9ZkPPydRCxVnyWS1khECFQxp5Ep3NbbKtxHNX5HeULzXN5q0EQO39UN6iBhiI34eZkH7PoAm3Vk5xns//FjTAvQw6wZUu8LwvZTaihs+upx2zZysq6CEBKoeNZqed9+Tf+qHow0P5pxmiu+or+DAgMBAAE=
    Spec:
      Docker:
        Entrypoint:
        - Rscript
        - Saturating-Forecasts.R
        - example_wp_log_R.csv
        - outputs/output0.pdf
        - outputs/output1.pdf
        Image: jsace/r-prophet
      Engine: Docker
      Language:
        JobContext: {}
      Publisher: Estuary
      Resources:
        GPU: ""
      Sharding:
        BatchSize: 1
        GlobPatternBasePath: /inputs
      Verifier: Noop
      Wasm: {}
      inputs:
      - CID: QmY8BAftd48wWRYDf5XnZGkhwqgjpzjyUG3hN1se6SYaFt
        StorageSource: IPFS
        path: /example_wp_log_R.csv
      outputs:
      - Name: outputs
        StorageSource: IPFS
        path: /outputs


If you see that the job has completed and there are no errors, then you can download the results with the following command:


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

    Fetching results of job '0316d0c2-162d-4c57-9c10-391c908f981d'...
    Results for job '0316d0c2-162d-4c57-9c10-391c908f981d' have been written to...
    results


After the download has finished you should 
see the following contents in results directory


```bash
%%bash
ls results/combined_results/outputs
```

    output0.pdf
    output1.pdf


You can't natively display PDFs in notebooks, so here are some static images of the PDFS:

* output0.pdf

![](output0.png)


* output1.pdf

![](output1.png)

