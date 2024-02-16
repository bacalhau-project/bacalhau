---
sidebar_label: Genomics
sidebar_position: 3
---
# Running Genomics on Bacalhau


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

# Introduction

Kipoi _(pronounce: kípi; from the Greek κήποι: gardens)_ is an API and a repository of ready-to-use trained models for genomics. It currently contains 2201 different models, covering canonical predictive tasks in transcriptional and post-transcriptional gene regulation. Kipoi's API is implemented as a [python package](https://github.com/kipoi/kipoi) and it is also accessible from the command line.

## TD;lR
Running genomics model on Bacalhau


## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Containerize your Script using Docker

To run Genomics on Bacalhau we need to set up a Docker container. To do this, you'll need to create a `Dockerfile` and add your desired configuration. The Dockerfile is a text document that contains the commands that specify how the image will be built.

```
FROM kipoi/kipoi-veff2:py37

RUN kipoi_veff2_predict ./examples/input/test.vcf ./examples/input/test.fa ./output.tsv -m "DeepSEA/predict" -s "diff" -s "logit"
```

### Build the container

The `docker build` command builds Docker images from a Dockerfile.

```
docker build -t <hub-user>/<repo-name>:<tag> .
```

Before running the command replace;

- **hub-user** with your docker hub username, If you don’t have a docker hub account [follow these instructions to create a Docker Account](https://docs.docker.com/docker-id/), and use the username of the account you created

- **repo-name** with the name of the container, you can name it anything you want

- **tag** this is not required but you can use the latest tag

In our case

```bash
docker build -t ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1 .
```

### Push the container

Next, upload the image to the registry. This can be done by using the Docker hub username, repo name or tag.

```
docker push <hub-user>/<repo-name>:<tag>
```

## Running a Bacalhau job to Generate Genomics Data

After the repo image has been pushed to Docker Hub, we can now use the container for running on Bacalhau. To submit a job, run the following Bacalhau command:



```bash
%%bash --out job_id
bacalhau docker run \
--id-only \
--wait \
--timeout 3600 \
--wait-timeout-secs 3600 \
jsacex/kipoi-veff2:py37 \
-- kipoi_veff2_predict ./examples/input/test.vcf ./examples/input/test.fa ../outputs/output.tsv -m "DeepSEA/predict" -s "diff" -s "logit"
```

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%env JOB_ID={job_id}
```


## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
```

When it says `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe ${JOB_ID}
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

After the download has finished you should see the following contents in the results directory

## Viewing your Job Output

To view the file, run the following command:


```bash
%%bash
ls results/ # list the contents of the current directory
cat results/outputs/output.tsv | head -n 10 # display the contents of the current directory
```
