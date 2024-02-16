---
sidebar_label: "Training-Pytorch-Model"
sidebar_position: 2
---
# Training Pytorch Model with Bacalhau


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

In this example tutorial, we will show you how to train a Pytorch RNN MNIST neural network model with Bacalhau. PyTorch is a framework developed by Facebook AI Research for deep learning, featuring both beginner-friendly debugging tools and a high level of customization for advanced users, with researchers and practitioners using it across companies like Facebook and Tesla. Applications include computer vision, natural language processing, cryptography, and more.

## TD;LR

Running any type of Pytorch model with Bacalhau

## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)



```python
!command -v bacalhau >/dev/null 2>&1 || (export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
path=!echo $PATH
%env PATH=./:{path[0]}
```

## Training the Model Locally

To train our model locally, we will start by cloning the Pytorch examples [repo](https://github.com/pytorch/examples)


```bash
%%bash
git clone https://github.com/pytorch/examples
```

Install the following


```bash
%%bash
pip install torch
```


```bash
%%bash
pip install torchvision
```

Next, we run the command below to begin the training of the _mnist_rnn_ model. We added the `--save-model` flag to save the model


```bash
%%bash
python ./examples/mnist_rnn/main.py --save-model
```

Next, the downloaded MNIST dataset is saved in the `data` folder.

## Uploading Dataset to IPFS

Now that we have downloaded our dataset, the next step is to upload it to IPFS. The simplest way to upload the data to IPFS is to use a third-party service to "pin" data to the IPFS network, to ensure that the data exists and is available. To do this you need an account with a pinning service like [web3.storage](https://web3.storage/) or [Pinata](https://pinata.cloud/) or [NFT.Storage](https://nft.storage/). Once registered you can use their UI or API or SDKs to upload files.

Once you have uploaded your data, you'll be finished copying the CID. Here is the dataset we have uploaded [https://gateway.pinata.cloud/ipfs/QmdeQjz1HQQdT9wT2NHX86Le9X6X6ySGxp8dfRUKPtgziw/?filename=data](https://gateway.pinata.cloud/ipfs/QmdeQjz1HQQdT9wT2NHX86Le9X6X6ySGxp8dfRUKPtgziw/?filename=data)


## Running a Bacalhau Job

After the repo image has been pushed to Docker Hub, we can now use the container for running on Bacalhau. To submit a job, run the following Bacalhau command:


```bash
%%bash --out job_id
bacalhau docker run \
--gpu 1 \
--timeout 3600 \
--wait-timeout-secs 3600 \
--wait \
--id-only \
pytorch/pytorch \
-w /outputs \
 -i ipfs://QmdeQjz1HQQdT9wT2NHX86Le9X6X6ySGxp8dfRUKPtgziw:/data \
-i https://raw.githubusercontent.com/pytorch/examples/main/mnist_rnn/main.py \
-- python ../inputs/main.py --save-model
```

### Structure  of the command

- `bacalhau docker run`: call to bacalhau

- `--gpu 1`: Request 1 GPU to train the model

- `pytorch/pytorch`: Using the official pytorch Docker image

- `-i ipfs://QmdeQjz1HQQd.....`: Mounting the uploaded dataset to the path

- `-i https://raw.githubusercontent.com/py..........`: Mounting our training script we will use the URL to this [Pytorch example](https://github.com/pytorch/examples/blob/main/mnist_rnn/main.py)

- `-w /outputs:` Our working directory is /outputs. This is the folder where we will save the model as it will automatically get uploaded to IPFS as outputs

`python ../inputs/main.py --save-model`: URL script gets mounted to the /inputs folder in the container.

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
cat results/stdout # displays the contents of the file given to it as a parameter.
ls results/outputs/ # list the successfully trained model
```
