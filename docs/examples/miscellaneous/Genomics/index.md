# Running Genomics on bacalhau


[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/Genomics/BIDS/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=miscellaneous/Genomics/index.ipynb)

# Introduction

Kipoi _(pronounce: kípi; from the Greek κήποι: gardens)_ is an API and a repository of ready-to-use trained models for genomics. It currently contains 2201 different models, covering canonical predictive tasks in transcriptional and post-transcriptional gene regulation. Kipoi's API is implemented as a [python package](https://github.com/kipoi/kipoi) and it is also accessible from the command line.


## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Running Genomics on Bacalhau using Docker

To run Genomics on Bacalhau we need to set up a Docker container. To do this, you'll need to:
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

### Running a Bacalhau job to Generate Genomics Data

To submit a job, run the following Bacalhau command:



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

When a job is sumbitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%%env JOB_ID={job_id}
```


## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`. 


```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 11:01:31 [0m[97;40m cf10a68c [0m[97;40m Docker jsacex/kipoi-... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmU3EV213QSHeK... [0m


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

    [90m11:03:34.094 |[0m [32mINF[0m [1mbacalhau/get.go:67[0m[36m >[0m Fetching results of job 'cf10a68c-9fb7-41fa-991b-a736cbf6277f'...
    2022/10/02 11:03:35 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.
    [90m11:03:45.277 |[0m [32mINF[0m [1mipfs/downloader.go:115[0m[36m >[0m Found 1 result shards, downloading to temporary folder.
    [90m11:09:55.538 |[0m [32mINF[0m [1mipfs/downloader.go:195[0m[36m >[0m Combining shard from output volume 'outputs' to final location: '/content/results'


After the download has finished you should see the following contents in results directory

## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**, **per_shard files**, and the **raw** directory. To view the file, run the following command:


```bash
%%bash
ls results/ # list the contents of the current directory ("
cat results/combined_results/outputs/output.tsv | head -n 10 #display the contents of the file given to it as a parameter.
```
