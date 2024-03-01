---
sidebar_label: "Gromacs"
sidebar_position: 4
---
# Running Gromacs on Bacalhau

## Introduction

GROMACS is a package for high-performance molecular dynamics and output analysis. Molecular dynamics is a computer simulation method for analyzing the physical movements of atoms and molecules

In this example, we will make use of [gmx pdb2gmx](https://manual.gromacs.org/documentation/current/onlinehelp/gmx-pdb2gmx.html#description) program to add hydrogens to the molecules and generates coordinates in Gromacs (Gromos) format and topology in Gromacs format.  

In this example tutorial, our focus will be on running Gromacs package with Bacalhau


## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)



## Downloading datasets

Datasets can be found here [https://www.rcsb.org](https://www.rcsb.org), In this example we use [RCSB PDB - 1AKI](https://www.rcsb.org/structure/1AKI) dataset. After downloading place it in a folder called “input”


```
input
└── 1AKI.pdb
```


## Uploading the datasets to IPFS

The simplest way to upload the data to IPFS is to use a third-party service to "pin" data to the IPFS network, to ensure that the data exists and is available. To do this you need an account with a pinning service like [NFT.storage](https://nft.storage/) or [Pinata](https://pinata.cloud/). Once registered you can use their UI or API or SDKs to upload files.

Alternatively, you can upload your dataset to IPFS using [IPFS CLI](https://docs.ipfs.tech/install/command-line/#official-distributions):


```
$ ipfs add -r input/
added QmTCCqPzX3qSJHuMeSma9uCqUnriZ5eJX7MnxebxydL89f input/1AKI.pdb
added QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9 input
 113.59 KiB / 113.59 KiB [============================================================================================] 100.00%
```

Copy the CID in the end which is `QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9 ` 



## Running Bacalhau Job

Let's run a Bacalhau job that converts coordinate files to topology and FF-compliant coordinate files:

```bash
%%bash --out job_id
bacalhau docker run \
    --id-only \
    --wait \
    --timeout 3600 \
    --wait-timeout-secs 3600 \
    -i ipfs://QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9:/input \
    gromacs/gromacs \
    -- /bin/bash -c 'echo 15 | gmx pdb2gmx -f input/1AKI.pdb -o outputs/1AKI_processed.gro -water spc'
```

### Structure of the command

Lets look closely at the command above:

`bacalhau docker run`: call to Bacalhau

`-i ipfs://QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9:/input`: here we mount the CID of the dataset we uploaded to IPFS to use on the job

`gromacs/gromacs`: we use the official [gromacs - Docker Image](https://hub.docker.com/r/gromacs/gromacs)

`gmx pdb2gmx`: command in GROMACS that performs the conversion of molecular structural data from the Protein Data Bank (PDB) format to the GROMACS format, which is used for conducting Molecular Dynamics (MD) simulations and analyzing the results. Additional parameters could be found here [gmx pdb2gmx — GROMACS 2022.2 documentation](https://manual.gromacs.org/documentation/current/onlinehelp/gmx-pdb2gmx.html)

`-f input/1AKI.pdb`: input file

`-o outputs/1AKI_processed.gro`: output file

`-water` Water model to use. In this case we use spc

:::tip
For a similar tutorial that you can try yourself, check out [KALP-15 in DPPC - GROMACS Tutorial](http://www.mdtutorials.com/gmx/membrane_protein/01_pdb2gmx.html)
:::

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

```python
%env JOB_ID={job_id}
```

## Checking the State of your Jobs

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

## Viewing your Job Output

To view the file, run the following command:


```bash
%%bash
cat results/outputs/1AKI_processed.gro  
```

## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).