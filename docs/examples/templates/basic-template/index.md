---
sidebar_label: "Basic Template"
sidebar_position: 1
---
# Title of Example

> Change the links of this icons:

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/templates/basic-template/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=templates/basic-template/index.ipynb)

This notebook is a basic example of using notebooks to create examples. It demonstrates some of ipython's basic features to achieve common Bacalhau tasks. Other more advanced templates are available in the [templates](..) directory.

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
    total 73640
    -rw-r--r-- 1 phil staff       47 Sep 16 10:16 Dockerfile
    -rw-r--r-- 1 phil staff       98 Sep 16 13:00 README.md
    -rwxr-xr-x 1 phil staff 75054546 Sep 16 13:01 bacalhau
    -rw-r--r-- 1 phil staff   137052 Sep 16 12:23 example-image.jpg
    -rw-r--r-- 1 phil staff   195696 Sep 16 12:59 index.ipynb
    -rw-r--r-- 1 phil staff       94 Sep 16 12:58 myfile.py
    -rw-r--r-- 1 phil staff       20 Sep 16 10:31 small-toy-dataset.csv



```python
!echo "But this works too, but remember the ! is rendered in the docs"
!curl https://ifconfig.me/
```

    But this works too, but remember the ! is rendered in the docs
    92.4.101.140

## Working with Bacalhau

> Remember that the user's and CI context likely won't have Bacalhau installed, so you need to install it.

Install Bacalhau with the following command and then hack the kernels PATH to include the installed location. This means we can use the `bacalhau` command as if someone had installed in globally


```python
!(export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
path=!echo $PATH
%env PATH=./:{path[0]}
```

    Your system is darwin_arm64
    
    BACALHAU CLI is detected:
    Client Version: v0.2.3
    Server Version: v0.2.3
    Reinstalling BACALHAU CLI - ./bacalhau...
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
    env: PATH=./:/Users/phil/.pyenv/versions/3.9.7/bin:/opt/homebrew/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/bin:/Users/phil/.gvm/bin:/opt/homebrew/opt/findutils/libexec/gnubin:/opt/homebrew/opt/coreutils/libexec/gnubin:/opt/homebrew/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/bin:/Users/phil/.pyenv/shims:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/MacGPG2/bin:/Users/phil/.nexustools



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

    12935a4b-c6c2-4ed3-b7da-c3b2f14941ab
    Note that bash is executed in a subprocess, so variables are only available within the same cell



```python
print("We can also do this with Python")
job_id = !bacalhau docker run --wait --wait-timeout-secs 100 ubuntu echo Hello World
```

    We can also do this with Python



```python
print("Which does work across cells", job_id[0])
```

    Which does work across cells b4246da6-b721-4c13-b32e-838a850eebd3



```python
!bacalhau list --id-filter {job_id[0]}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 12:03:08 [0m[97;40m b4246da6 [0m[97;40m Docker ubuntu echo H... [0m[97;40m Published [0m[97;40m          [0m[97;40m /ipfs/bafybeidu4zm6w... [0m


## Working With Images

You can either dump an image right in markdown like this:

![](example-image.jpg)

Or resultant images can be displayed in the notebook using the `Image` class from `IPython.display`. You can also use the `display` function to display other objects.


```python
import IPython.display as display
display.Image("example-image.jpg")
```




    
![jpeg](index_files/index_12_0.jpg)
    



## Working With Raw Text Files

When working with raw text files like Dockerfiles, be sure to show these to the user.


```python
%cat Dockerfile
```

    FROM example-dockerfile
    RUN echo "do something"

You can even write files directly from your notebook for later use...


```python
%%writefile myfile.py

print("This is code in a newly created python file. Use %%writefile -a to append to files.")
```

    Overwriting myfile.py



```python
%run -i 'myfile.py'
```

    This is code in a newly created python file. Use %%writefile -a to append to files.


## Working With Files

If your file is small, fine, shove it in git. But if it's big, use the production GCS bucket for http-accessible public data or IPFS, whichever makes more sense.

To access files in GCS, you can use the `gsutil` command line tool. You can also use the `gcsfs` library to access GCS from Python. You'll need to make sure you have the correct credentials to access the bucket. This can be done by executing `(cd ops/terraform; bash scripts/connect_workspace.sh production)` from the root of the Bacalhau repository.

When uploading files, please use the same directory structure as this repository to keep things organised. For example, I uploaded a small-toy-dataset.csv using:

```
gsutil cp templates/basic-template/small-toy-dataset.csv gs://bacalhau-examples/templates/basic-template/small-toy-example.csv
```


```bash
%%bash
curl -s https://storage.googleapis.com/bacalhau-examples/templates/basic-template/small-toy-example.csv
```

    a,very,small,dataset
