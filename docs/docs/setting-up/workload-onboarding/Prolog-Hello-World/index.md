---
sidebar_label: "Prolog Script"
sidebar_position: 4
---
# Running a Prolog Script


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

## Introduction
Prolog is intended primarily as a declarative programming language: the program logic is expressed in terms of relations, represented as facts and rules. A computation is initiated by running a query over these relations.
Prolog is well-suited for specific tasks that benefit from rule-based logical queries such as searching databases, voice control systems, and filling templates.

This tutorial is a quick guide on how to run a hello world script on Bacalhau.

## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)


## 1. Running Locally​


To get started, install swipl



```bash
%%bash
sudo add-apt-repository ppa:swi-prolog/stable
sudo apt-get update
sudo apt-get install swi-prolog
```

Create a file called `helloworld.pl`. The following script prints ‘Hello World’ to the stdout:



```python
%%writefile helloworld.pl
hello_world :- write('Hello World'), nl,
               halt.
```

Running the script to print out the output:



```bash
%%bash
swipl -q -s helloworld.pl -g hello_world
```

After the script has run successfully locally we can now run it on Bacalhau.

Before running it on Bacalhau we need to upload it to IPFS.

Using the `IPFS cli`



```python
!wget https://dist.ipfs.io/go-ipfs/v0.4.2/go-ipfs_v0.4.2_linux-amd64.tar.gz
!tar xvfz go-ipfs_v0.4.2_linux-amd64.tar.gz
!mv go-ipfs/ipfs /usr/local/bin/ipfs
!ipfs init
!ipfs cat /ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme
!ipfs config Addresses.Gateway /ip4/127.0.0.1/tcp/8082
!ipfs config Addresses.API /ip4/127.0.0.1/tcp/5002
!nohup ipfs daemon > startup.log &
```

Run the command below to check if our script has been uploaded.


```python
!ipfs add helloworld.pl
```

This command outputs the CID. Copy the CID of the file, which in our case is `QmYq9ipYf3vsj7iLv5C67BXZcpLHxZbvFAJbtj7aKN5qii`

Since the data uploaded to IPFS isn’t pinned, we will need to do that manually. Check this information on how to pin your [data](../../../setting-up/data-ingestion/pin.md) We recommend using [NFT.Storage](https://nft.storage/).



## 2. Running a Bacalhau Job


We will mount the script to the container using the `-i` flag:
 `-i: ipfs://< CID >:/< name-of-the-script >`.

To submit a job, run the following Bacalhau command:


```bash
%%bash --out job_id
bacalhau docker run \
    -i ipfs://QmYq9ipYf3vsj7iLv5C67BXZcpLHxZbvFAJbtj7aKN5qii:/helloworld.pl \
    --wait \
    --id-only \
    swipl \
    -- swipl -q -s helloworld.pl -g hello_world
```

### Structure of the Command


`-i ipfs://QmYq9ipYf3vsj7iLv5C67BXZcpLHxZbvFAJbtj7aKN5qii:/helloworld.pl` : Sets the input data for the container. `QmYq9ipYf3vsj7iLv5C67BXZcpLHxZbvFAJbtj7aKN5qii` is our CID which points to the `helloworld.pl` file on the IPFS network. This file will be accessible within the container.

`-- swipl -q -s helloworld.pl -g hello_world`: instructs SWI-Prolog to load the program from the `helloworld.pl` file and execute the `hello_world` function in quiet mode:

`-q`: running in quiet mode

`-s`: load file as a script. In this case we want to run the `helloworld.pl` script

`-g`: is the name of the function you want to execute. In this case its `hello_world`

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on:

```python
%env JOB_ID={job_id}
```

## 3. Checking the State of your Jobs

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

## 4. Viewing your Job Output

To view the file, run the following command:


```bash
%%bash
cat results/stdout
```

## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).
