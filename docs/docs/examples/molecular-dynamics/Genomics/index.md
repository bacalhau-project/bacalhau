---
sidebar_label: Genomics
sidebar_position: 3
---
# Running Genomics on Bacalhau

## Introduction

Kipoi _(pronounce: kípi; from the Greek κήποι: gardens)_ is an API and a repository of ready-to-use trained models for genomics. It currently contains 2201 different models, covering canonical predictive tasks in transcriptional and post-transcriptional gene regulation. Kipoi's API is implemented as a [python package](https://github.com/kipoi/kipoi), and it is also accessible from the command line.

In this tutorial example, we will run a genomics model on Bacalhau.


## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)  


## Running Locally​

To run locally you need to install kipoi-veff2. You can find out the information about installing and usage [here](https://github.com/kipoi/kipoi-veff2/blob/main/README.md)
 

In our case this will be the following command:

```bash
kipoi_veff2_predict ./examples/input/test.vcf ./examples/input/test.fa ./output.tsv -m "DeepSEA/predict" -s "diff" -s "logit"
```

## Containerize Script using Docker

To run Genomics on Bacalhau we need to set up a Docker container. To do this, you'll need to create a `Dockerfile` and add your desired configuration. The Dockerfile is a text document that contains the commands that specify how the image will be built.

```
FROM kipoi/kipoi-veff2:py37

RUN kipoi_veff2_predict ./examples/input/test.vcf ./examples/input/test.fa ./output.tsv -m "DeepSEA/predict" -s "diff" -s "logit"
```
We will use the `kipoi/kipoi-veff2:py37` image and perform variant-centered effect prediction using the `kipoi_veff2_predict` tool.

:::info
See more information on how to containerize your script/app [here](https://docs.docker.com/get-started/02_our_app/)
:::


### Build the container

The `docker build` command builds Docker images from a Dockerfile.

```
docker build -t <hub-user>/<repo-name>:<tag> .
```

Before running the command replace:

**`hub-user`** with your docker hub username. If you don’t have a docker hub account [follow these instructions to create a Docker Account](https://docs.docker.com/docker-id/), and use the username of the account you created

**`repo-name`** with the name of the container, you can name it anything you want

**`tag`** this is not required but you can use the latest tag

In our case

```bash
docker build -t jsacex/kipoi-veff2:py37 .
```

### Push the container

Next, upload the image to the registry. This can be done by using the Docker hub username, repo name or tag.

```
docker push <hub-user>/<repo-name>:<tag>
```

## Running a Bacalhau job

After the repo image has been pushed to Docker Hub, we can now use the container for running on Bacalhau. To submit a job for generating genomics data, run the following Bacalhau command:



```bash
%%bash --out job_id
bacalhau docker run \
<<<<<<< HEAD
    --id-only \
    --memory 20Gb \
    --wait \
    --timeout 3600 \
    --wait-timeout-secs 3600 \
    jsacex/kipoi-veff2:py37 \
    -- kipoi_veff2_predict ./examples/input/test.vcf ./examples/input/test.fa ../outputs/output.tsv -m "DeepSEA/predict" -s "diff" -s "logit"
=======
--id-only \
--wait \
--timeout 3600 \
--wait-timeout-secs 3600 \
jsacex/kipoi-veff2:py37 \
-- kipoi_veff2_predict ./examples/input/test.vcf ./examples/input/test.fa ../outputs/output.tsv -m "DeepSEA/predict" -s "diff" -s "logit"
>>>>>>> main
```

### Structure of the command

Let's look closely at the command above:

`bacalhau docker run`: call to Bacalhau

`jsacex/kipoi-veff2:py37`: the name of the image we are using

`kipoi_veff2_predict ./examples/input/test.vcf ./examples/input/test.fa ../outputs/output.tsv -m "DeepSEA/predict" -s "diff" -s "logit"`: the command that will be executed inside the container. It performs variant-centered effect prediction using the kipoi_veff2_predict tool

`./examples/input/test.vcf`: the path to a Variant Call Format (VCF) file containing information about genetic variants

`./examples/input/test.fa`: the path to a FASTA file containing DNA sequences. FASTA files contain nucleotide sequences used for variant effect prediction

`../outputs/output.tsv`: the path to the output file where the prediction results will be stored. The output file format is Tab-Separated Values (TSV), and it will contain information about the predicted variant effects

`-m "DeepSEA/predict"`: specifies the model to be used for prediction

`-s "diff" -s "logit"`: indicates using two scoring functions for comparing prediction results. In this case, the "diff" and "logit" scoring functions are used. These scoring functions can be employed to analyze differences between predictions for the reference and alternative alleles.


When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%env JOB_ID={job_id}
```


## Checking the State of your Jobs

**Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
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
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

## Viewing your Job Output

To view the file, run the following command:


```bash
%%bash
<<<<<<< HEAD
cat results/outputs/output.tsv | head -n 10  
=======
ls results/ # list the contents of the current directory
cat results/outputs/output.tsv | head -n 10 # display the contents of the current directory
>>>>>>> main
```

## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).