---
sidebar_label: "Training-Pytorch-Model"
sidebar_position: 2
---
# Training Pytorch Model with Bacalhau

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/model-training/Training-Tensorflow-Model/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=model-training/Training-Tensorflow-Model/index.ipynb)

In this example tutorial, we will show you how to train a Pytorch RNN MNIST neural network model with Bacalhau. PyTorch is a framework developed by Facebook AI Research for deep learning, featuring both beginner-friendly debugging tools and a high-level of customization for advanced users, with researchers and practitioners using it across companies like Facebook and Tesla. Applications include computer vision, natural language processing, cryptography, and more

## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)


## Training the Model Locally

To train our model locally, we will start by cloning the Pytorch examples [repo](https://github.com/pytorch/examples)


```bash
%%bash
git clone https://github.com/pytorch/examples
```

Next, we run the command below to begin training of the _mnist_rnn_ model. We added the `--save-model` flag to save the model


```bash
%%bash
python ./examples/mnist_rnn/main.py --save-model
```

Next, we will download the MNIST dataset by creating a folder `data` where we will save the downloaded dataset


```bash
%%bash
mkdir ../data
```

If you inspect the code [here](https://github.com/pytorch/examples/blob/main/mnist_rnn/main.py) you'll see the folder referenced in the code. Here is the a small section of the code that references the folder


```python
    train_loader = torch.utils.data.DataLoader(
        datasets.MNIST('../data', train=True, download=True,
                       transform=transforms.Compose([
                           transforms.ToTensor(),
                           transforms.Normalize((0.1307,), (0.3081,))
                       ])),
        batch_size=args.batch_size, shuffle=True, **kwargs)
    test_loader = torch.utils.data.DataLoader(
        datasets.MNIST('../data', train=False, transform=transforms.Compose([
            transforms.ToTensor(),
            transforms.Normalize((0.1307,), (0.3081,))
        ])),
        batch_size=args.test_batch_size, shuffle=True, **kwargs)
```

## Uploading the dataset to IPFS

Now that we have downloaded our dataset, the next step is to upload it to IPFS. This can be done using the IPFS CLI


```
ipfs add -r data
```

Since the data Uploaded To IPFS using IPFS CLI isn’t pinned or will be garbage collected. The data needs to be **pinned**. Pinning is the mechanism that allows you to tell IPFS to always keep a given object somewhere, the default being your local node, though this can be different if you use a third-party remote pinning service.

There a different pinning services available you can you any one of them

### Pinata

You can use [Pinata](https://app.pinata.cloud/) to save data on IPFS node. Once you have uploaded your data to Pinata, you'll finished copy the CID

### [NFT.Storage](https://nft.storage/) (Recommneded Option)

[NFT.Storage](https://nft.storage/) is a recommneded option. To upload your dataset using NFTup just drag and drop your directory it will upload it to IPFS. See more information [here](https://nft.storage/docs/how-to/nftup/) 

You can view you uploaded dataset by clicking on the Gateway URL [https://gateway.pinata.cloud/ipfs/QmdeQjz1HQQdT9wT2NHX86Le9X6X6ySGxp8dfRUKPtgziw/?filename=data](https://gateway.pinata.cloud/ipfs/QmdeQjz1HQQdT9wT2NHX86Le9X6X6ySGxp8dfRUKPtgziw/?filename=data)

## Running a Bacalhau Job to Generate a Trained Model

After the repo image has been pushed to docker hub, we can now use the container for running on Bacalhau. To submit a job, run the following Bacalhau command:


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
 -v QmdeQjz1HQQdT9wT2NHX86Le9X6X6ySGxp8dfRUKPtgziw:/data \
-u https://raw.githubusercontent.com/pytorch/examples/main/mnist_rnn/main.py \
-- python ../inputs/main.py --save-model
```

### Sturucture of the command

- `--gpu 1`: Request 1 GPU to train the model

- `pytorch/pytorch`: Using the official pytorch Docker image

- `-v QmdeQjz1HQQd.....`: Mounting the uploaded dataset to path

- `-u https://raw.githubusercontent.com/py..........`: Mounting our training script we will use the URL to this [Pytorch example](https://github.com/pytorch/examples/blob/main/mnist_rnn/main.py) 

- `-w /outputs:` Our working directory is /outputs. This is the folder where we will to save the model as it will automatically gets uploaded to IPFS as outputs

`python ../inputs/main.py --save-model`: URL script gets mounted to the /inputs folder in the container. 

When a job is sumbitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

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

After the download has finished you should see the following contents in results directory

## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**, **per_shard files**, and the **raw** directory. To view the file, run the following command:


```bash
%%bash
ls results/ # list the contents of the current directory 
cat results/combined_results/stdout # displays the contents of the file given to it as a parameter.
ls results/combined_results/outputs/ # list the successfully trained model
```
