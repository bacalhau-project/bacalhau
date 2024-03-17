---
sidebar_label: "Huggingface-Model-Inference"
sidebar_position: 1
---
# Running Inference on Dolly 2.0 Model with Hugging Face

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/model-inference/Huggingface-Model-Inference/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=model-inference/Huggingface-Model-Inference/index.ipynb)
[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

## Introduction
Dolly 2.0, the groundbreaking, open-source, instruction-following Large Language Model (LLM) that has been fine-tuned on a human-generated instruction dataset, licensed for both research and commercial purposes. Developed using the EleutherAI Pythia model family, this 12-billion-parameter language model is built exclusively on a high-quality, human-generated instruction following dataset, contributed by Databricks employees.

Dolly 2.0 package is open source, including the training code, dataset, and model weights, all available for commercial use. This unprecedented move empowers organizations to create, own, and customize robust LLMs capable of engaging in human-like interactions, without the need for API access fees or sharing data with third parties.

## Running locally
### Prerequisites
- A NVIDIA GPU
- Python

### Installing dependencies



```bash
%%bash
pip -q install git+https://github.com/huggingface/transformers # need to install from github
pip -q install accelerate>=0.12.0
```

Creating the inference script



```python
%%writefile inference.py
import argparse
import torch
from transformers import pipeline

def main(prompt_string, model_version):

    # use dolly-v2-12b if you're using Colab Pro+, using pythia-2.8b for Free Colab
    generate_text = pipeline(model=model_version,
                            torch_dtype=torch.bfloat16,
                            trust_remote_code=True,
                            device_map="auto")

    print(generate_text(prompt_string))

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--prompt", type=str, required=True, help="The prompt to be used in the GPT model")
    parser.add_argument("--model_version", type=str, default="./databricks/dolly-v2-12b", help="The model version to be used")
    args = parser.parse_args()
    main(args.prompt, args.model_version)

```

## Building the container (optional)

### Prerequisite
- Install Docker on your local machine.
- Sign up for a DockerHub account if you don't already have one.
Steps

Step 1: Create a Dockerfile
Create a new file named Dockerfile in your project directory with the following content:
```Dockerfile
FROM huggingface/transformers-pytorch-deepspeed-nightly-gpu

RUN apt-get update -y

RUN pip -q install git+https://github.com/huggingface/transformers

RUN pip -q install accelerate>=0.12.0

WORKDIR /

# COPY ./dolly_inference.py .
```
This Dockerfile sets up a container with the necessary dependencies and installs the Segment Anything Model from its GitHub repository.

Step 2: Build the Docker Image
In your terminal, navigate to the directory containing the Dockerfile and run the following command to build the Docker image:

```bash
docker build -t your-dockerhub-username/sam:lite .
```
Replace your-dockerhub-username with your actual DockerHub username. This command will build the Docker image and tag it with your DockerHub username and the name "sam".

Step 3: Push the Docker Image to DockerHub
Once the build process is complete, Next, push the Docker image to DockerHub using the following command:
```bash
docker push your-dockerhub-username/sam:lite
```

Again, replace your-dockerhub-username with your actual DockerHub username. This command will push the Docker image to your DockerHub repository.

## Running Inference on Bacalhau

### Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

### Structure of the command


```
bacalhau docker run \
--gpu 1 \
-w /inputs \
-i gitlfs://huggingface.co/databricks/dolly-v2-3b.git \
-i https://gist.githubusercontent.com/js-ts/d35e2caa98b1c9a8f176b0b877e0c892/raw/3f020a6e789ceef0274c28fc522ebf91059a09a9/inference.py \
jsacex/dolly_inference:latest \
 -- python inference.py --prompt "Where is Earth located ?" --model_version "./databricks/dolly-v2-3b"
 ```




* `docker run`: Docker command to run a container from a specified image.

* `--gpu 1`: Flag to specify the number of GPUs to use for the execution. In this case, 1 GPU will be used.

* `-w /inputs`: Flag to set the working directory inside the container to `/inputs`.

* `-i gitlfs://huggingface.co/databricks/dolly-v2-3b.git`: Flag to clone the Dolly V2-3B model from Hugging Face's repository using Git LFS. The files will be mounted to `/inputs/databricks/dolly-v2-3b`.

* `-i https://gist.githubusercontent.com/js-ts/d35e2caa98b1c9a8f176b0b877e0c892/raw/3f020a6e789ceef0274c28fc522ebf91059a09a9/inference.py`: Flag to download the `inference.py` script from the provided URL. The file will be mounted to `/inputs/inference.py`.

* `jsacex/dolly_inference:latest`: The name and the tag of the Docker image.

* The command to run inference on the model: `python inference.py --prompt "Where is Earth located ?" --model_version "./databricks/dolly-v2-3b"`.

  * `inference.py`: The Python script that runs the inference process using the Dolly V2-3B model.

  * `--prompt "Where is Earth located ?"`: Specifies the text prompt to be used for the inference.

  * `--model_version "./databricks/dolly-v2-3b"`: Specifies the path to the Dolly V2-3B model. In this case, the model files are mounted to `/inputs/databricks/dolly-v2-3b`.
