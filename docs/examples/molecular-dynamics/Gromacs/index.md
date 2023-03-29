# Molecular Dynamics with Bacalhau


[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/Gromacs/BIDS/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=miscellaneous/Gromacs/index.ipynb)
[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

## Introduction

GROMACS is a package for high-performance molecular dynamics and output analysis.

Molecular dynamics is a computer simulation method for analyzing the physical movements of atoms and molecules

In this example we will make use of [gmx pdb2gmx](https://manual.gromacs.org/documentation/current/onlinehelp/gmx-pdb2gmx.html#description) program to add hydrogens to the molecules and generates coordinates in Gromacs (Gromos) format and a topology in Gromacs format



### **Downloading datasets**

Datasets can be found here [https://www.rcsb.org](https://www.rcsb.org), In this example we use [RCSB PDB - 1AKI](https://www.rcsb.org/structure/1AKI) dataset. After downloading place it in a folder called “input”


```
input
└── 1AKI.pdb
```


### **Uploading the datasets to IPFS**

Upload the directory to IPFS using IPFS CLI ([Installation Instructions](https://docs.ipfs.tech/install/command-line/#official-distributions)) [Not recommended]


```
$ ipfs add -r input/
added QmTCCqPzX3qSJHuMeSma9uCqUnriZ5eJX7MnxebxydL89f input/1AKI.pdb
added QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9 input
 113.59 KiB / 113.59 KiB [============================================================================================] 100.00%
```

Copy the CID in the end which is `QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9 `

Upload the directory to IPFS using [Pinata](https://app.pinata.cloud/) (Recommended)

Click on the upload folder button and select the datasets folder that you want to upload
![](https://i.imgur.com/TfNP9Lv.png)

After the Upload has finished copy the CID (highlighted part)

![](https://i.imgur.com/WO6QlN4.png)



#### **Running the command on Bacalhau**

This command converts coordinate files to topology and FF-compliant coordinate files:

```
bacalhau docker run \
-v QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9:/input \
gromacs/gromacs \
-- /bin/bash -c 'echo 15 | gmx pdb2gmx -f input/1AKI.pdb -o outputs/1AKI_processed.gro -water spc'
```
Lets look at the command above more closely:

* `bacalhau docker run` using the docker backend

* `-v QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9:/input` here we mount the CID of the dataset we uploaded to IPFS and mount it to a folder called data on the container

* `gromacs/gromacs` we use the official [gromacs - Docker Image](https://hub.docker.com/r/gromacs/gromacs) 

* `-f input/1AKI.pdb` input file

* `-o output/1AKI_processed.gro` output file

* `-water` Water model to use in this case we use spc

Additional parameters could be found here [gmx pdb2gmx — GROMACS 2022.2 documentation](https://manual.gromacs.org/documentation/current/onlinehelp/gmx-pdb2gmx.html) 

(similar tutorial you can try yourself [KALP-15 in DPPC - GROMACS Tutorial](http://www.mdtutorials.com/gmx/membrane_protein/01_pdb2gmx.html) )


Installing Bacalhau


```bash
%%bash
curl -sL https://get.bacalhau.org/install.sh | bash
```

    Your system is linux_amd64
    No BACALHAU detected. Installing fresh BACALHAU CLI...
    Getting the latest BACALHAU CLI...
    Installing v0.2.3 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.2.3/bacalhau_v0.2.3_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.2.3/bacalhau_v0.2.3_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into /usr/local/bin successfully.
    Client Version: v0.2.3
    Server Version: v0.2.3



```bash
%%bash --out job_id
bacalhau docker run \
--id-only \
--wait \ 
--timeout 3600 \
--wait-timeout-secs 3600 \
-v QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9:/input \
gromacs/gromacs
-- /bin/bash -c 'echo 15 | gmx pdb2gmx -f input/1AKI.pdb -o outputs/1AKI_processed.gro -water spc'
```


```python
%env JOB_ID={job_id}
```


Running the commands will output a UUID. This is the ID of the job that was created. You can check the status of the job with the following command:



```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
```


Where it says `Completed`, that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```bash
%%bash
bacalhau describe ${JOB_ID}
```

To Download the results of your job, run the following command:


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

    [90m12:19:36.609 |[0m [32mINF[0m [1mbacalhau/get.go:67[0m[36m >[0m Fetching results of job 'ab354ccc-f02e-4262-ad0b-f33ec78803cc'...
    2022/09/18 12:19:37 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.
    [90m12:19:47.364 |[0m [32mINF[0m [1mipfs/downloader.go:115[0m[36m >[0m Found 1 result shards, downloading to temporary folder.
    [90m12:19:51.091 |[0m [32mINF[0m [1mipfs/downloader.go:195[0m[36m >[0m Combining shard from output volume 'outputs' to final location: '/content/results'


After the download has finished you should 
see the following contents in results directory


```bash
%%bash
ls results/
```

    shards	stderr	stdout	volumes


Rach repository contains selfexplanatory results.
