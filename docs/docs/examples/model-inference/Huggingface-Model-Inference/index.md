---
sidebar_label: "Inference on Dolly 2.0 Model with Hugging Face"
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
1. A NVIDIA GPU
1. Python

### Installing dependencies



```bash
pip -q install git+https://github.com/huggingface/transformers # need to install from github
pip -q install accelerate # ensure you are using version higher than 0.12.0
```

Create an `inference.py` file with following code:

```python
# content of the inference.py file
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

You may want to create your own container for this kind of task. In that case, use the instructions for [creating](https://docs.docker.com/get-started/02_our_app/) and [publishing](https://docs.docker.com/get-started/04_sharing_app/) your own image in the docker hub. Use ` huggingface/transformers-pytorch-deepspeed-nightly-gpu` as base image, install dependencies listed above and copy the `inference.py` into it.


## Running Inference on Bacalhau

### Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)

### Structure of the command

1. `export JOB_ID=$( ... )`: Export results of a command execution as environment variable
1. `bacalhau docker run`: Run a job using docker executor.
1. `--gpu 1`: Flag to specify the number of GPUs to use for the execution. In this case, 1 GPU will be used.
1. `-w /inputs`: Flag to set the working directory inside the container to `/inputs`.
1. `-i gitlfs://huggingface.co/databricks/dolly-v2-3b.git`: Flag to clone the Dolly V2-3B model from Hugging Face's repository using Git LFS. The files will be mounted to `/inputs/databricks/dolly-v2-3b`.
1. `-i https://gist.githubusercontent.com/js-ts/d35e2caa98b1c9a8f176b0b877e0c892/raw/3f020a6e789ceef0274c28fc522ebf91059a09a9/inference.py`: Flag to download the `inference.py` script from the provided URL. The file will be mounted to `/inputs/inference.py`.
1. `jsacex/dolly_inference:latest`: The name and the tag of the Docker image.
1. The command to run inference on the model: `python inference.py --prompt "Where is Earth located ?" --model_version "./databricks/dolly-v2-3b"`. It consists of:
    1. `inference.py`: The Python script that runs the inference process using the Dolly V2-3B model.
    1. `--prompt "Where is Earth located ?"`: Specifies the text prompt to be used for the inference.
    1. `--model_version "./databricks/dolly-v2-3b"`: Specifies the path to the Dolly V2-3B model. In this case, the model files are mounted to `/inputs/databricks/dolly-v2-3b`.

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

```bash
export JOB_ID=$(bacalhau docker run \
    --gpu 1 \
    --id-only \
    -w /inputs \
    -i gitlfs://huggingface.co/databricks/dolly-v2-3b.git \
    -i https://gist.githubusercontent.com/js-ts/d35e2caa98b1c9a8f176b0b877e0c892/raw/3f020a6e789ceef0274c28fc522ebf91059a09a9/inference.py \
    jsacex/dolly_inference:latest \
    -- python inference.py --prompt "Where is Earth located ?" --model_version "./databricks/dolly-v2-3b")
 ```

 ### Checking the State of your Jobs

1. **Job status**: You can check the status of the job using `bacalhau list`:


```bash
bacalhau list --id-filter ${JOB_ID}
```

When it says `Completed`, that means the job is done, and we can get the results.

2. **Job information**: You can find out more information about your job by using `bacalhau describe`:

```bash
bacalhau describe ${JOB_ID}
```

3. **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
rm -rf results && mkdir results
bacalhau get ${JOB_ID} --output-dir results
```

### Viewing your Job Output

After the download has finished we can see the results in the `results/outputs` folder.