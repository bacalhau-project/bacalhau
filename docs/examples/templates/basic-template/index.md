---
sidebar_label: "Basic Template"
sidebar_position: 1
---
# Title of Example

This notebook is a basic example of using notebooks to create examples. It demonstrates some of ipython's basic features to achieve common Bacalhau tasks. Other more advanced templates are available in the [templates](../templates) directory.

### Why a Notebook?

Notebooks are great because:

* Readers can "execute" the documentation. They can not only read the example, but try it!
* They are testable. We can run the notebook in CI and check that it works.
* They are interactive. Readers can alter the notebooks and play with the code.
* They are great for hackathons. Just point people to the notebook and get hacking.

### What Happens to This Notebook

There are a few CI scripts that operate on notebooks:
* Whenever you push to the main branch on this repository, a github action will automatically render your ipynb's into markdown and push them to the [docs repository](https://github.com/bacalhau-project/docs.bacalhau.org/).
* Whenever you push, pytest will run to ensure that all notebooks execute without error.

### Key Requirements

* We use Python as the kernel for our notebooks. This is the most common kernel and is used in our tests.
* Large datafiles should not be stored in the repo. Store data in either the GCS bucket or IPFS (depending on the example).
* Make sure all cells run successfully to pass tests. If a cell takes a very long time you may want to skip tests. See below.

### Structure

All examples must exist within a directory. The notebook should be called `index.ipynb` (so that it gets rendered as the index.html page in the docs). You may have other supporting files in the directory. For example:

```
basic-template
├── Dockerfile
├── README.md
├── example-image.jpg
├── index.ipynb
└── small-toy-dataset.csv
```

### Metadata

You can control some elements of the documentation rendering by adding metadata to the notebook. This is done by adding a raw cell at the top of the notebook. The metadata is in YAML format. See the top of this file for an example.

## Working With Bash

The following demonstrates how to work with bash commands in a notebook. Note that these commands execute within the context of your kernel. If you don't have the required dependencies installed, you will need to install them first. Read more about [working with bash in notebooks](https://ipython.readthedocs.io/en/stable/interactive/magics.html#magic-bash).

```bash


```bash
%%bash
echo "This is one way of working with bash, which is good because it renders nicely in the documentation"
ls -l
```

    This is one way of working with bash, which is good because it renders nicely in the documentation
    total 144
    -rw-r--r-- 1 phil staff      0 Sep 16 09:38 Dockerfile
    -rw-r--r-- 1 phil staff      0 Sep 16 09:38 README.md
    -rw-r--r-- 1 phil staff 137052 Sep 16 09:42 example-image.jpg
    -rw-r--r-- 1 phil staff   4192 Sep 16 09:52 index.ipynb
    -rw-r--r-- 1 phil staff      0 Sep 16 09:38 small-toy-dataset.csv



```python
!echo "But this works too, but remember the ! is rendered in the docs"
!curl https://ifconfig.me/
```

    But this works too, but remember the ! is rendered in the docs
    92.4.101.140

## Working with Bacalhau

> Remember that the user's and CI context likely won't have Bacalhau installed, so you need to install it.

Install Bacalhau with the following command:


```bash
%%bash
(export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
```

    Your system is darwin_arm64
    No BACALHAU detected. Installing fresh BACALHAU CLI...
    Getting the latest BACALHAU CLI...
    Installing v0.2.3 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.2.3/bacalhau_v0.2.3_darwin_arm64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.2.3/bacalhau_v0.2.3_darwin_arm64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into . successfully.
    Client Version: v0.2.3
    Server Version: v0.2.3



```bash
%%bash
bacalhau version
```

    Client Version: v0.2.3
    Server Version: v0.2.3



```bash
%%bash
job_id=$(bacalhau docker run ubuntu echo Hello World)
echo $job_id
echo "Note that bash is executed in a subprocess, so variables are only available within the same cell"
```

    d5464534-76f9-44a6-af23-8699a830ed72
    Note that bash is executed in a subprocess, so variables are only available within the same cell



```python
print("We can also do this with Python")
job_id = !bacalhau docker run --wait --wait-timeout-secs 100 ubuntu echo Hello World

```

    We can also do this with Python



```python
print("Which does work across cells", job_id[0])
```

    Which does work across cells 1a6f8588-9eb6-40b6-a73c-7abd8da0f876



```python
!bacalhau list --id-filter {job_id[0]}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 09:11:05 [0m[97;40m 1a6f8588 [0m[97;40m Docker ubuntu echo H... [0m[97;40m Published [0m[97;40m          [0m[97;40m /ipfs/bafybeidu4zm6w... [0m



```python

```
