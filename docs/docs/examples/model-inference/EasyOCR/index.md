---
sidebar_label: "EasyOCR"
sidebar_position: 8
---

# EasyOCR (Optical Character Recognition) on Bacalhau


## Introduction

In this example tutorial, we use Bacalhau and Easy OCR to digitize paper records or for recognizing characters or extract text data from images stored on IPFS, S3 or on the web. [EasyOCR](https://www.jaided.ai/) is a ready-to-use OCR with 80+ supported languages and all popular writing scripts including Latin, Chinese, Arabic, Devanagari, Cyrillic and etc. With easy OCR you use the pre-trained models or use your own fine-tuned model.

## TL;DR
```bash
bacalhau docker run \
    -i ipfs://bafybeibvcllzpfviggluobcfassm3vy4x2a4yanfxtmn4ir7olyzfrgq64:/root/.EasyOCR/model/zh_sim_g2.pth  \
    -i https://raw.githubusercontent.com/JaidedAI/EasyOCR/ae773d693c3f355aac2e58f0d8142c600172f016/examples/chinese.jpg \
    --timeout 3600 \
    --wait-timeout-secs 3600 \
    --gpu 1  \
    --memory 10Gb \
    --cpu 3 \
    --id-only \
    --wait \
    jsacex/easyocr \
    --  easyocr -l ch_sim  en -f ./inputs/chinese.jpg --detail=1 --gpu=True
```


## Running Easy OCR Locally​

Install the required dependencies


```bash
pip install --upgrade easyocr
```

Load the different example images


```bash
npx degit JaidedAI/EasyOCR/examples -f
```

List all the images. You'll see a output like this:


```bash
ls -l

total 3508
-rw-r--r-- 1 root root   59898 Jun 16 22:36 chinese.jpg
-rw-r--r-- 1 root root   97910 Jun 16 22:36 easyocr_framework.jpeg
-rw-r--r-- 1 root root 1740957 Jun 16 22:36 english.png
-rw-r--r-- 1 root root  487995 Jun 16 22:36 example2.png
-rw-r--r-- 1 root root  127454 Jun 16 22:36 example3.png
-rw-r--r-- 1 root root  488641 Jun 16 22:36 example.png
-rw-r--r-- 1 root root  168376 Jun 16 22:36 french.jpg
-rw-r--r-- 1 root root   42159 Jun 16 22:36 japanese.jpg
-rw-r--r-- 1 root root  225531 Jun 16 22:36 korean.png
drwxr-xr-x 1 root root    4096 Jun 15 13:37 sample_data
-rw-r--r-- 1 root root   82229 Jun 16 22:36 thai.jpg
-rw-r--r-- 1 root root   34706 Jun 16 22:36 width_ths.png
```

Next, we create a reader to do OCR to get coordinates which represent a rectangle containing text and the text itself:


```python
import easyocr
reader = easyocr.Reader(['th','en'])
# Doing OCR. Get bounding boxes.
bounds = reader.readtext('thai.jpg')
bounds
```

## Containerize your Script using Docker

:::tip
You can skip this step and go straight to running a [Bacalhau job](#running-a-bacalhau-job-to-generate-easy-ocr-output)
:::

We will use the `Dockerfile` that is already created in the [Easy OCR repo](https://github.com/JaidedAI/EasyOCR). Use the command below to clone the repo

```
git clone https://github.com/JaidedAI/EasyOCR
cd EasyOCR
```

### Build the Container

The `docker build` command builds Docker images from a Dockerfile.

```bash
docker build -t <hub-user>/<repo-name>:<tag> .
```

Before running the command replace:

1. **hub-user** with your docker hub username, If you don’t have a docker hub account follow [these instructions](https://docs.docker.com/docker-id/) to create docker account, and use the username of the account you created
1. **repo-name** with the name of the container, you can name it anything you want
1. **tag** this is not required but you can use the latest tag

### Push the container

Next, upload the image to the registry. This can be done by using the Docker hub username, repo name, or tag.

```
docker push <hub-user>/<repo-name>:<tag>
```

## Running a Bacalhau Job to Generate Easy OCR output

### Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)

Now that we have an image in the docker hub (your own or an example image from the manual), we can use the container for running on Bacalhau.
### Structure of the imperative command


Let's look closely at the command below:

1. `export JOB_ID=$( ... )` exports the job ID as environment variable
1. `bacalhau docker run`: call to bacalhau
1. The `--gpu 1` flag is set to specify hardware requirements, a GPU is needed to run such a job
1. The `--id-only` flag is set to print only job id
1. `-i ipfs://bafybeibvc......` Mounts the model from IPFS
1. `-i https://raw.githubusercontent.com...` Mounts the Input Image from a URL
1. `jsacex/easyocr` the name and the tag of the docker image we are using
1. `--  easyocr -l ch_sim  en -f ./inputs/chinese.jpg --detail=1 --gpu=True` execute script with following paramters:
    1. `-l ch_sim`: the name of the model 
    1. `-f ./inputs/chinese.jpg`: path to the input Image or directory
    1. `--detail=1`: level of detail
    1. `--gpu=True`: we set this flag to true since we are running inference on a GPU. If you run this on a CPU - set this flag to false

Since the model and the image aren't present in the container we will mount the image from an URL and the model from IPFS. You can find models to download from [here](https://www.jaided.ai/easyocr/modelhub/). You can choose the model you want to use in this case we will be using the `zh_sim_g2` model


```bash
export JOB_ID=$(bacalhau docker run \
    -i ipfs://bafybeibvcllzpfviggluobcfassm3vy4x2a4yanfxtmn4ir7olyzfrgq64:/root/.EasyOCR/model/zh_sim_g2.pth  \
    -i https://raw.githubusercontent.com/JaidedAI/EasyOCR/ae773d693c3f355aac2e58f0d8142c600172f016/examples/chinese.jpg \
    --timeout 3600 \
    --wait-timeout-secs 3600 \
    --gpu 1  \
    --memory 10Gb \
    --cpu 3 \
    --id-only \
    --wait \
    jsacex/easyocr \
    --  easyocr -l ch_sim  en -f ./inputs/chinese.jpg --detail=1 --gpu=True)
```


When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

### Declarative job description

The same job can be presented in the [declarative](../../../setting-up/jobs/job-specification/job.md) format. In this case, the description will look like this:

```yaml
name: EasyOCR
type: batch
count: 1
tasks:
  - name: My main task
    Engine:
      type: docker
      params:
        Image: "jsacex/easyocr" 
        Entrypoint:
          - /bin/bash
        Parameters:
          - -c
          - easyocr -l ch_sim  en -f ./inputs/chinese.jpg --detail=1 --gpu=True
    InputSources:
    - Target: "/inputs/chinese.jpg"
      Source:
      Type: "urlDownload"
        Params:
          URL: "https://raw.githubusercontent.com/JaidedAI/EasyOCR/ae773d693c3f355aac2e58f0d8142c600172f016/examples/chinese.jpg"
    - Target: "/root/.EasyOCR/model/zh_sim_g2.pth"
      Source:
        Type: "s3"
        Params:
          Bucket: "landsat-image-processing"
          Key: "*"
          Region: "us-east-1"
    Resources:
      GPU: "1"
```

The job description should be saved in `.yaml` format, e.g. `easyocr.yaml`, and then run with the command:
```bash
bacalhau job run easyocr.yaml
```

## Checking the State of your Jobs

### Job status

You can check the status of the job using `bacalhau list`.


```bash
bacalhau list --id-filter ${JOB_ID}
```

When it says `Completed`, that means the job is done, and we can get the results.

### Job information

You can find out more information about your job by using `bacalhau describe`.


```bash
bacalhau describe ${JOB_ID}
```

### Job download

You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

After the download has finished you should see the following contents in results directory

## Viewing your Job Output

Now you can find the file in the `results/outputs` folder. You can view results by running following commands:


```bash
ls results # list the contents of the current directory 
```
```bash
cat results/stdout # displays the contents of the current directory 
```
