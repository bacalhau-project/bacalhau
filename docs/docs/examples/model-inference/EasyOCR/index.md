---
sidebar_label: "EasyOCR"
sidebar_position: 8
---

# EasyOCR (Optical Character Recognition) on Bacalhau

[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

## Introduction

In this example tutorial, we use Bacalhau and Easy OCR to digitize paper records or for recognizing characters or extract text data from images stored on IPFS/Filecoin or on the web. [EasyOCR](https://www.jaided.ai/) is a ready-to-use OCR with 80+ supported languages and all popular writing scripts including Latin, Chinese, Arabic, Devanagari, Cyrillic and etc. With easy OCR you use the pre-trained models or use your own fine-tuned model.

## TD:LR
Using Bacalhau and Easy OCR to extract text data from images stored on the web.

## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Running Easy OCR Locally​

Install the required dependencies


```bash
%%bash
pip install easyocr
```

Load the different example images


```bash
%%bash
npx degit JaidedAI/EasyOCR/examples -f
```

List all the images


```bash
%%bash
ls -l
```

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


To displaying an image from the list


```python
# show an image
import PIL
from PIL import ImageDraw
im = PIL.Image.open("thai.jpg")
```

Next, we create a reader to do OCR to get coordinates which represent a rectangle containing text and the text itself


```python
# If you change to GPU instance, it will be faster. But CPU is enough.
# (by MENU > Runtime > Change runtime type > GPU, then redo from beginning )
import easyocr
reader = easyocr.Reader(['th','en'])
# Doing OCR. Get bounding boxes.
bounds = reader.readtext('thai.jpg')
bounds
```

## Containerize your Script using Docker

:::tip
You can skip this step and go straight to running a Bacalhau job
:::

We will use the `Dockerfile` that is already created in the [Easy OCR repo](https://github.com/JaidedAI/EasyOCR). Use the command below to clone the repo

```
git clone https://github.com/JaidedAI/EasyOCR
cd EasyOCR
```

### Build the Container

The `docker build` command builds Docker images from a Dockerfile.

```
docker build -t hub-user/repo-name:tag .
```

Before running the command replace;

- **hub-user** with your docker hub username, If you don’t have a docker hub account [follow these instructions to create docker account](https://docs.docker.com/docker-id/), and use the username of the account you created

- **repo-name** with the name of the container, you can name it anything you want

- **tag** this is not required but you can use the latest tag

### Push the container

Next, upload the image to the registry. This can be done by using the Docker hub username, repo name, or tag.

```
docker push <hub-user>/<repo-name>:<tag>
```

## Running a Bacalhau Job to Generate Easy OCR output

After the repo image has been pushed to Docker Hub, we can now use the container for running on Bacalhau. To submit a job, run the following Bacalhau command:


```bash
%%bash --out job_id
bacalhau docker run \
-i ipfs://bafybeibvcllzpfviggluobcfassm3vy4x2a4yanfxtmn4ir7olyzfrgq64:/root/.EasyOCR/model/zh_sim_g2.pth  \
-i https://raw.githubusercontent.com/JaidedAI/EasyOCR/ae773d693c3f355aac2e58f0d8142c600172f016/examples/chinese.jpg \
--timeout 3600 \
--wait-timeout-secs 3600 \
--gpu 1  \
--id-only \
--wait \
jsacex/easyocr \
--  easyocr -l ch_sim  en -f ./inputs/chinese.jpg --detail=1 --gpu=True

```

Since the model and the image aren't present in the container we will mount the image from an URL and the model from IPFS. You can find models to download from [here](https://www.jaided.ai/easyocr/modelhub/). You can choose the model you want to use in this case we will be using the zh_sim_g2 model

### Structure of the command

-  `-i ipfs://bafybeibvc......`: Mounting the model from IPFS

- `-i https://raw.githubusercontent.com.........` Mounting the Input Image from a URL
- `--gpu 1`: Specifying the no of GPUs

- `jsacex/easyocr`: Name of the Docker image

Breaking up the easyocr command

**--  easyocr -l ch_sim  en -f ./inputs/chinese.jpg --detail=1 --gpu=True**

- `-l`: the name of the model which is ch_sim

- `-f`: path to the input Image or directory

- `--detail=1`:  level of detail

- `--gpu=True`: we set this flag to true since we are running inference on a GPU, if you run this on a CPU you set this to false


When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
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
cat results/stdout # displays the contents of the current directory
```
