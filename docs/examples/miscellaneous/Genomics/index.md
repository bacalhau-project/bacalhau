# Running Genomics on bacalhau


[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/Genomics/BIDS/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=miscellaneous/Genomics/index.ipynb)

# Introduction

Kipoi _(pronounce: kípi; from the Greek κήποι: gardens)_ is an API and a repository of ready-to-use trained models for genomics. It currently contains 2201 different models, covering canonical predictive tasks in transcriptional and post-transcriptional gene regulation. Kipoi's API is implemented as a [python package](https://github.com/kipoi/kipoi) and it is also accessible from the command line.

## Setting up Docker

To set up Docker, you'll need to:
- Create a `Dockerfile`. The Dockerfile is a text document that contains the commands used to assemble the image.
- Add your desired configuration to the `Dockerfile`. These commands specify how the image will be built, and what extra requirements will be included.
```
FROM kipoi/kipoi-veff2:py37

RUN kipoi_veff2_predict ./examples/input/test.vcf ./examples/input/test.fa ./output.tsv -m "DeepSEA/predict" -s "diff" -s "logit"
```
- Next, we will use the `python:3.8` docker image to build the docker container to download the models and weights. Before running the command below, replace:
    - `hub-user` with your docker hub username. If you don’t have a docker hub account follow these [instructions](https://docs.docker.com/docker-id/) to create docker account and use the username of the account you created

    - `repo-name` with the name of the container, you can name it anything you want

    - `tag` with the latest tag (optional)

```
docker build -t <hub-user>/<repo-name>:<tag>
```
- Push the repository to the designated registry in Docker hub by using its name or tag.

```
 docker push <hub-user>/<repo-name>:<tag>
```
After the repo image has been pushed to docker hub, we can now use run the container on Bacalhau

## Running the Container on Bacalhau

To run your Docker container on Bacalhau. First;

- Install bacalhau

```bash
curl -sL https://get.bacalhau.org/install.sh | bash
```

- To get your Bacalhau job id, run the following command:

```bash
bacalhau docker run \
--id-only \
--wait \ 
--timeout 3600 \
--wait-timeout-secs 3600 \
jsacex/kipoi-veff2:py37 \
-- kipoi_veff2_predict ./examples/input/test.vcf ./examples/input/test.fa ../outputs/output.tsv -m "DeepSEA/predict" -s "diff" -s "logit"
```
Running the command above will output a UUID (like `54506541-4eb9-45f4-a0b1-ea0aecd34b3e`). This is the ID of the job that was created. 

You can use an enviromental variable to store your Job ID

```python
%env JOB_ID={job_id}
```

## Checking the State of your Jobs

- **Job status**: You can check the status of the job with the following command:

```bash
bacalhau list --id-filter ${JOB_ID} --wide
```
When it says `Published`, that means the job is done, and we can get the results. 

- **Job information**: To find out more information about your job, run the following command:

```bash
bacalhau describe ${JOB_ID}
```                          

When there is no error, the state of our job  will be complete which means you can download the results.

- **Download job results**: You can download your job results directly by using `bacalhau get`. You can also choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.

```bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```
After the download has finished you should see the following contents in _results_ directory.

## Viewing your Job Output

To view your output, run the following command:  

```bash
ls results/
```
Each job creates 3 subfolders: the **combined_results**, **per_shard** files, and the **raw** directory. 

In each of these sub_folders, you'll find the **studout** and **stderr** file.

To view the file in the _stdout_ folder, run the following command:

```bash
cat results/job-id/combined_results/stdout
```
