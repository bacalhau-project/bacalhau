---
sidebar_label: "Stable Diffusion Dreambooth"
sidebar_position: 1
---
# Stable Diffusion Dreambooth (Finetuning)


## Introduction

Stable diffusion has revolutionalized text2image models by producing high quality images based on a prompt. Dreambooth is a approach for personalization of text-to-image diffusion models. With images as input subject, we can fine-tune a pretrained text-to-image model

Although the [dreambooth paper](https://arxiv.org/abs/2208.12242) used [Imagen](https://imagen.research.google/) to finetune the pre-trained model since both the Imagen model and Dreambooth code are closed source, several opensource projects have emerged using stable diffusion.

Dreambooth makes stable-diffusion even more powered with the ability to generate realistic looking pictures of humans, animals or any other object by just training them on 20-30 images. 

In this example tutorial, we will be fine-tuning a pretrained stable diffusion using images of a human and generating images of him drinking coffee.

## TL;DR
The following command generates the following:
- **Subject**: SBF
- **Prompt**: a photo of SBF without hair

```
bacalhau docker run \
 --gpu 1 \
 --timeout 3600 \
 --wait-timeout-secs 3600 \
  -i ipfs://QmRKnvqvpFzLjEoeeNNGHtc7H8fCn9TvNWHFnbBHkK8Mhy  \
  jsacex/dreambooth:full \
  -- bash finetune.sh /inputs /outputs "a photo of sbf man" "a photo of man" 3000 "/man" "/model"
```


### Inference
```
bacalhau docker run \
 --gpu 1 \
  -i ipfs://QmUEJPr5pfV6tRzWQuNSSb3wdcN6tRQS5tdk3dYSCJ55Xs:/SBF.ckpt \
   jsacex/stable-diffusion-ckpt \
   -- conda run --no-capture-output -n ldm python scripts/txt2img.py --prompt "a photo of sbf without hair" --plms --ckpt ../SBF.ckpt --skip_grid --n_samples 1 --skip_grid --outdir ../outputs 
```
Output:

![png](index_files/sd-dreambooth-tldr-image.png)



## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)


## Setting up Docker Container

:::info
You can skip this section entirely and directly go to [running a job on Bacalhau](#run-the-bacalhau-job-on-the-fine-tuned-model)
:::

Building this container requires you to have a supported GPU which needs to have 16gb+ of memory, since it can be resource intensive.

We will create a  `Dockerfile` and add the desired configuration to the file. Following commands specify how the image will be built, and what extra requirements will be included:


```Dockerfile
FROM pytorch/pytorch:1.12.1-cuda11.3-cudnn8-devel

WORKDIR /

# Install requirements
# RUN git clone https://github.com/TheLastBen/diffusers

RUN apt update && apt install wget git unzip -y

RUN wget -q https://gist.githubusercontent.com/js-ts/28684a7e6217214ec944a9224584e9af/raw/d7492bc8f36700b75d51e3346259d1a466b99a40/train_dreambooth.py

RUN wget -q https://github.com/TheLastBen/diffusers/raw/main/scripts/convert_diffusers_to_original_stable_diffusion.py

# RUN cp /content/convert_diffusers_to_original_stable_diffusion.py /content/diffusers/scripts/convert_diffusers_to_original_stable_diffusion.py

RUN pip install -qq git+https://github.com/TheLastBen/diffusers

RUN pip install -q accelerate==0.12.0 transformers ftfy bitsandbytes gradio natsort

# Install xformers

RUN pip install -q https://github.com/metrolobo/xformers_wheels/releases/download/1d31a3ac_various_6/xformers-0.0.14.dev0-cp37-cp37m-linux_x86_64.whl

RUN wget 'https://github.com/TheLastBen/fast-stable-diffusion/raw/main/Dreambooth/Regularization/Women' -O woman.zip

RUN wget 'https://github.com/TheLastBen/fast-stable-diffusion/raw/main/Dreambooth/Regularization/Men' -O man.zip

RUN wget 'https://github.com/TheLastBen/fast-stable-diffusion/raw/main/Dreambooth/Regularization/Mix' -O mix.zip

RUN unzip -j woman.zip -d woman

RUN unzip -j man.zip -d man

RUN unzip -j mix.zip -d mix
```

This container is using the `pytorch/pytorch:1.12.1-cuda11.3-cudnn8-devel` image and the working directory is set. Next, we add our custom code and pull the dependent repositories.

```bash
# finetune.sh
python clear_mem.py

accelerate launch train_dreambooth.py \
  --image_captions_filename \
  --train_text_encoder \
  --save_n_steps=$(expr $5 / 6) \
  --stop_text_encoder_training=$(expr $5 + 100) \
  --class_data_dir="$6" \
  --pretrained_model_name_or_path=${7:=/model} \
  --tokenizer_name=${7:=/model}/tokenizer/ \
  --instance_data_dir="$1" \
  --output_dir="$2" \
  --instance_prompt="$3" \
  --class_prompt="$4" \
  --seed=96576 \
  --resolution=512 \
  --mixed_precision="fp16" \
  --train_batch_size=1 \
  --gradient_accumulation_steps=1 \
  --use_8bit_adam \
  --learning_rate=2e-6 \
  --lr_scheduler="polynomial" \
  --center_crop \
  --lr_warmup_steps=0 \
  --max_train_steps=$5

echo  Convert weights to ckpt
python convert_diffusers_to_original_stable_diffusion.py --model_path $2  --checkpoint_path $2/model.ckpt --half
echo model saved at $2/model.ckpt
```

The shell script is there to make things much simpler since the command to train the model needs many parameters to pass and later convert the model weights to the checkpoint, you can edit this script and add in your own parameters

### Downloading the models

To download the models and run a test job in the Docker file, copy the following:

```Dockerfile
FROM pytorch/pytorch:1.12.1-cuda11.3-cudnn8-devel
WORKDIR /
# Install requirements
# RUN git clone https://github.com/TheLastBen/diffusers
RUN apt update && apt install wget git unzip -y
RUN wget -q https://gist.githubusercontent.com/js-ts/28684a7e6217214ec944a9224584e9af/raw/d7492bc8f36700b75d51e3346259d1a466b99a40/train_dreambooth.py
RUN wget -q https://github.com/TheLastBen/diffusers/raw/main/scripts/convert_diffusers_to_original_stable_diffusion.py
# RUN cp /content/convert_diffusers_to_original_stable_diffusion.py /content/diffusers/scripts/convert_diffusers_to_original_stable_diffusion.py
RUN pip install -qq git+https://github.com/TheLastBen/diffusers
RUN pip install -q accelerate==0.12.0 transformers ftfy bitsandbytes gradio natsort
# Install xformers
RUN pip install -q https://github.com/metrolobo/xformers_wheels/releases/download/1d31a3ac_various_6/xformers-0.0.14.dev0-cp37-cp37m-linux_x86_64.whl
# You need to accept the model license before downloading or using the Stable Diffusion weights. Please, visit the [model card](https://huggingface.co/runwayml/stable-diffusion-v1-5), read the license and tick the checkbox if you agree. You have to be a registered user in 🤗 Hugging Face Hub, and you'll also need to use an access token for the code to work.
# https://huggingface.co/settings/tokens
RUN mkdir -p ~/.huggingface
RUN echo -n "<your-hugging-face-token>" > ~/.huggingface/token
# copy the test dataset from a local file
# COPY jfk /jfk

# Download and extract the test dataset
RUN wget https://github.com/js-ts/test-images/raw/main/jfk.zip
RUN unzip -j jfk.zip -d jfk
RUN mkdir model
RUN wget 'https://github.com/TheLastBen/fast-stable-diffusion/raw/main/Dreambooth/Regularization/Women' -O woman.zip
RUN wget 'https://github.com/TheLastBen/fast-stable-diffusion/raw/main/Dreambooth/Regularization/Men' -O man.zip
RUN wget 'https://github.com/TheLastBen/fast-stable-diffusion/raw/main/Dreambooth/Regularization/Mix' -O mix.zip
RUN unzip -j woman.zip -d woman
RUN unzip -j man.zip -d man
RUN unzip -j mix.zip -d mix

RUN  accelerate launch train_dreambooth.py \
  --image_captions_filename \
  --train_text_encoder \
  --save_starting_step=5\
  --stop_text_encoder_training=31 \
  --class_data_dir=/man \
  --save_n_steps=5 \
  --pretrained_model_name_or_path="CompVis/stable-diffusion-v1-4" \
  --instance_data_dir="/jfk" \
  --output_dir="/model" \
  --instance_prompt="a photo of jfk man" \
  --class_prompt="a photo of man" \
  --instance_prompt="" \
  --seed=96576 \
  --resolution=512 \
  --mixed_precision="fp16" \
  --train_batch_size=1 \
  --gradient_accumulation_steps=1 \
  --use_8bit_adam \
  --learning_rate=2e-6 \
  --lr_scheduler="polynomial" \
  --center_crop \
  --lr_warmup_steps=0 \
  --max_train_steps=30

COPY finetune.sh /finetune.sh

RUN wget -q https://gist.githubusercontent.com/js-ts/624fecc3fff807d4948688cb28993a94/raw/fd69ac084debe26a815485c1f363b8a45566f1ba/clear_mem.py
# Removing your token
RUN rm -rf  ~/.huggingface/token
```

Then execute `finetune.sh` with following commands:
```bash
# finetune.sh
python clear_mem.py

accelerate launch train_dreambooth.py \
    --image_captions_filename \
   --train_text_encoder \
    --save_n_steps=$(expr $5 / 6) \
    --stop_text_encoder_training=$(expr $5 + 100) \
       --class_data_dir="$6" \
  --pretrained_model_name_or_path=${7:=/model} \
--tokenizer_name=${7:=/model}/tokenizer/ \
    --instance_data_dir="$1" \
    --output_dir="$2" \
    --instance_prompt="$3" \
   --class_prompt="$4" \
    --seed=96576 \
    --resolution=512 \
    --mixed_precision="fp16" \
    --train_batch_size=1 \
    --gradient_accumulation_steps=1 \
    --use_8bit_adam \
    --learning_rate=2e-6 \
    --lr_scheduler="polynomial" \
    --center_crop \
    --lr_warmup_steps=0 \
    --max_train_steps=$5

echo  Convert weights to ckpt
python convert_diffusers_to_original_stable_diffusion.py --model_path $2  --checkpoint_path $2/model.ckpt --half
echo model saved at $2/model.ckpt
```

### Build the Docker container

We will run `docker build` command to build the container:

```
docker build -t <hub-user>/<repo-name>:<tag> .
```

Before running the command replace:

1. **hub-user** with your docker hub username, If you don’t have a docker hub account follow [these instructions](https://docs.docker.com/docker-id/) to create a Docker account, and use the username of the account you create. 
1. **repo-name** with the name of the container, you can name it anything you want. 
1. **tag** this is not required but you can use the latest tag


Now you can push this repository to the registry designated by its name or tag.

```
docker push <hub-user>/<repo-name>:<tag>
```

### Create the Subject Dataset

The optimal dataset size is between 20-30 images. You can choose the images of the subject in different positions, full body images, half body, pictures of the face etc.

Only the subject should appear in the image so you can crop the image to just fit the subject. Make sure that the images are 512x512 size and are named in the following pattern: 

```
Subject Name.jpg, Subject Name (2).jpg ... Subject Name (n).jpg
```

You can view the [Subject Image dataset of David Aronchick](https://bafybeidqbuphwkqwgrobv2vakwsh3l6b4q2mx7xspgh4l7lhulhc3dfa7a.ipfs.nftstorage.link) for reference.

After the Subject dataset is created we upload it to IPFS.

## Uploading the Subject Images to IPFS

In this case, we will be using [NFT.Storage](https://nft.storage/) (Recommended Option) to upload files and directories with [NFTUp](https://nft.storage/docs/how-to/nftup/).

To upload your dataset using NFTup just drag and drop your directory it will upload it to IPFS:

![](https://i.imgur.com/g3VM2Kp.png)

After the checkpoint file has been uploaded, copy its CID which will look like this:

```bash
bafybeidqbuphwkqwgrobv2vakwsh3l6b4q2mx7xspgh4l7lhulhc3dfa7a
```

## Approaches to run a Bacalhau Job on a Finetuned Model

Since there are a lot of combinations that you can try, processing of finetuned model can take almost 1hr+ to complete. Here are a few approaches that you can try based on your requirements:


### Case 1: If the subject is of class male

#### Structure of the command

1. `bacalhau docker run`: call to bacalhau
1. The `--gpu 1` flag is set to specify hardware requirements, a GPU is needed to run such a job
1. `-i ipfs://bafybeidqbuphwkqwgrobv2vakwsh3l6b4q2mx7xspgh4l7lhulhc3dfa7a` Mounts the data from IPFS via its CID
1. `jsacex/dreambooth:latest` Name and tag of the docker image we are using
1. `-- bash finetune.sh /inputs /outputs "a photo of David Aronchick man" "a photo of man" 3000 "/man"` execute script with following paramters:
    1. `/inputs` Path to the subject Images 
    1. `/outputs` Path to save the generated outputs 
    1. `"a photo of < name of the subject > < class >"` -> `"a photo of David Aronchick man"` Subject name along with class   
    1. `"a photo of < class >`" -> `"a photo of man"` Name of the class   


```bash
bacalhau docker run \
  --gpu 1 \
  --timeout 3600 \
  --wait-timeout-secs 3600 \
  --timeout 3600 \
  --wait-timeout-secs 3600 \
  -i <CID-OF-THE-SUBJECT> \
  jsacex/dreambooth:full \
  -- bash finetune.sh /inputs /outputs "a photo of <name-of-the-subject> man" "a photo of man" 3000 "/man" "/model"
```

The number of iterations is 3000. This number should be no of subject images x 100. So if there are 30 images, it would be 3000. It takes around 32 minutes on a `v100` for 3000 iterations, but you can increase/decrease the number based on your requirements.


Here is our command with our parameters replaced:

```bash
bacalhau docker run \
  --gpu 1 \
  --timeout 3600 \
  --wait-timeout-secs 3600 \
  --timeout 3600 \
  --wait-timeout-secs 3600 \
  -i ipfs://bafybeidqbuphwkqwgrobv2vakwsh3l6b4q2mx7xspgh4l7lhulhc3dfa7a \
  --wait \
  --id-only \
  jsacex/dreambooth:full \
  -- bash finetune.sh /inputs /outputs "a photo of David Aronchick man" "a photo of man" 3000 "/man" "/model"
```

If your subject fits the above class, but has a different name you just need to replace the input CID and the subject name.

### Case 2 : If the subject is of class female

Use the `/woman` class images

```
bacalhau docker run \
  --gpu 1 \
  --timeout 3600 \
  --wait-timeout-secs 3600 \
  -i <CID-OF-THE-SUBJECT> \
  jsacex/dreambooth:full \
  -- bash finetune.sh /inputs /outputs "a photo of <name-of-the-subject> woman" "a photo of woman" 3000 "/woman"  "/model"
```

### Case 3: If the subject is of class mix

Here you can provide your own regularization images or use the mix class.

Use the `/mix` class images if the class of the subject is mix

```
bacalhau docker run \
  --gpu 1 \
  --timeout 3600 \
  --wait-timeout-secs 3600 \
  -i <CID-OF-THE-SUBJECT> \
  jsacex/dreambooth:full \
  -- bash finetune.sh /inputs /outputs "a photo of <name-of-the-subject> mix" "a photo of mix" 3000 "/mix"  "/model"
```

### Case 4: If you want a different tokenizer, model, and a different shell script with custom parameters

You can upload the model to IPFS and then create a gist, mount the model and script to the lightweight container

```
bacalhau docker run \
  --gpu 1 \
  --timeout 3600 \
  --wait-timeout-secs 3600 \
  -i ipfs://bafybeidqbuphwkqwgrobv2vakwsh3l6b4q2mx7xspgh4l7lhulhc3dfa7a:/aronchick \
  -i ipfs://<CID-OF-THE-MODEL>:/model 
  -i https://gist.githubusercontent.com/js-ts/54b270a36aa3c35fdc270640680b3bd4/raw/7d8e8fa47bc3811ef63772f7fc7f4360aa9d51a8/finetune.sh
  --wait \
  --id-only \
  jsacex/dreambooth:lite \
  -- bash /inputs/finetune.sh /aronchick /outputs "a photo of aronchick man" "a photo of man" 3000 "/man" "/model"
```

When a job is submitted, Bacalhau prints out the related `job_id`. Use the `export JOB_ID=$(bacalhau docker run ...)` wrapper to store that in an environment variable so that we can reuse it later on.

### Declarative job description

The same job can be presented in the [declarative](../../../setting-up/jobs/job-specification/job.md) format. In this case, the description will look like this. Change the command in the Parameters section and CID to suit your goals.

```yaml
name: Stable Diffusion Dreambooth Finetuning
type: batch
count: 1
tasks:
  - name: My main task
    Engine:
      type: docker
      params:
        Image: "jsacex/dreambooth:full" 
        Parameters:
          - bash finetune.sh /inputs /outputs "a photo of aronchick man" "a photo of man" 3000 "/man" "/model"
    InputSources:
      - Source:
          Type: "ipfs"
          Params:
            CID: "QmRKnvqvpFzLjEoeeNNGHtc7H8fCn9TvNWHFnbBHkK8Mhy"
        - Target: "/data"
    Resources:
      GPU: "1"
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


In the next steps, we will be doing inference on the finetuned model

## Inference on the Fine-Tuned Model

:::info

Refer to our [guide on CKPT model](../../model-inference/Stable-Diffusion-CKPT-Inference/) for more details of how to build a SD inference container
:::

Bacalhau currently doesn't support mounting subpaths of the CID, so instead of just mounting the `model.ckpt` file we need to mount the whole output CID which is 6.4GB, which might result in errors like `FAILED TO COPY /inputs`. So you have to manually copy the CID of the `model.ckpt` which is of 2GB.

To get the CID of the `model.ckpt` file go to `https://gateway.ipfs.io/ipfs/< YOUR-OUTPUT-CID >/outputs/`. For example:

```bash
https://gateway.ipfs.io/ipfs/QmcmD7M7pYLP8QgwjqpbP4dojRLiLuEBdhevuCD9kFmbdV/outputs/
```

If you use the [Brave](https://brave.com/) browser, you can use following:

```bash
ipfs://QmdpsqZn9BZx9XxzCsyPcJyS7yfYacmQXZxHzcuYwzmtGg/outputs
```

Or you can use the IPFS CLI:

```bash
ipfs ls QmdpsqZn9BZx9XxzCsyPcJyS7yfYacmQXZxHzcuYwzmtGg/outputs
```

![image](https://imgur.com/S2WSZTA.png)

Copy the link of `model.ckpt` highlighted in the box:
```bash
https://gateway.ipfs.io/ipfs/QmdpsqZn9BZx9XxzCsyPcJyS7yfYacmQXZxHzcuYwzmtGg?filename=model.ckpt
```
Then extract the CID portion of the link and copy it.

### Run the Bacalhau Job on the Fine-Tuned Model

To run a Bacalhau Job on the fine-tuned model, we will use the `bacalhau docker run` command.


```bash
export JOB_ID=$(bacalhau docker run \
  --gpu 1 \
  --timeout 3600 \
  --wait-timeout-secs 3600 \
  --wait \
  --id-only \
  -i ipfs://QmdpsqZn9BZx9XxzCsyPcJyS7yfYacmQXZxHzcuYwzmtGg \
  jsacex/stable-diffusion-ckpt \
  -- conda run --no-capture-output -n ldm python scripts/txt2img.py --prompt "a photo of aronchick drinking coffee" --plms --ckpt ../inputs/model.ckpt --skip_grid --n_samples 1 --skip_grid --outdir ../outputs)
```

If you are facing difficulties using the above method you can mount the whole output CID

```bash
export JOB_ID=$(bacalhau docker run \
  --gpu 1 \
  --timeout 3600 \
  --wait-timeout-secs 3600 \
  --wait \
  --id-only \
  -i ipfs://QmcmD7M7pYLP8QgwjqpbP4dojRLiLuEBdhevuCD9kFmbdV \
  jsacex/stable-diffusion-ckpt \
  -- conda run --no-capture-output -n ldm python scripts/txt2img.py --prompt "a photo of aronchick drinking coffee" --plms --ckpt ../inputs/outputs/model.ckpt --skip_grid --n_samples 1 --skip_grid --outdir ../outputs)
```

When a job is sumbitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

To check the status of your job and download results refer back to the [guide above](#checking-the-state-of-your-jobs).

We got an image like this as a result:

![png](index_files/index_31_0.png)