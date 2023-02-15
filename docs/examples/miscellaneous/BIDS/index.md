# Running BIDS Apps on bacalhau


# Introduction

What is a BIDS App? ([source](https://bids-apps.neuroimaging.io/about/))

A BIDS App is a container image capturing a neuroimaging pipeline that takes a BIDS formatted dataset as input. BIDS (Brain Imaging Data Structure) is an emerging standard for organizing and describing neuroimaging datasets. Each BIDS App has the same core set of command line arguments, making them easy to run and integrate into automated platforms. BIDS Apps are constructed in a way that does not depend on any software outside of the image other than the container engine.

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/miscellaneous/BIDS/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=miscellaneous/BIDS/index.ipynb)


## **Downloading datasets**

You can find the bids datasets in this google drive folder [archives](https://drive.google.com/drive/folders/0B2JWN60ZLkgkMGlUY3B4MXZIZW8?resourcekey=0-EYVSOlRbxeFKO8NpjWWM3w) 

download the relevant data, [ds005.tar](https://drive.google.com/drive/folders/0B2JWN60ZLkgkMGlUY3B4MXZIZW8), and untar it in a directory. `ds005` will be our input directory in the following example.


```
data
└── ds005
```




### **Uploading the datasets to IPFS**

Upload the directory to IPFS using IPFS CLI ([Installation Instructions](https://docs.ipfs.tech/install/command-line/#official-distributions))


```
$ ipfs add -r data
added QmdsFcNbja8vbeNEj6HGfbvJmuu3cXUmgV4CR3HQqNqsNK data/ds005/CHANGES
                                    .
                                    .
                                    .
added QmdnMxSSvD8QYR6F4S7wkgQsW16bR6U7zyDTbiEm72RPpB data/ds005
added QmaNyzSpJCt1gMCQLd3QugihY6HzdYmA8QMEa45LDBbVPz data
 1.77 GiB / 1.77 GiB [=========================================================================================] 100.00%
```


Copy the CID in the end which is `QmaNyzSpJCt1gMCQLd3QugihY6HzdYmA8QMEa45LDBbVPz`

Upload the directory to IPFS using [Pinata](https://app.pinata.cloud/) (Recommended)

Click on the upload folder button and select the bids datasets folder that you want to upload

![](https://i.imgur.com/btnHw3N.png)


After the Upload has finished copy the CID (highlighted part)

![](https://i.imgur.com/rETHXXz.png)





```python
!mkdir data
!wget https://dist.ipfs.io/go-ipfs/v0.4.2/go-ipfs_v0.4.2_linux-amd64.tar.gz
!tar xvfz go-ipfs_v0.4.2_linux-amd64.tar.gz
!mv go-ipfs/ipfs /usr/local/bin/ipfs
!cd data
!ipfs init
!ipfs cat /ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme
!ipfs config Addresses.Gateway /ip4/127.0.0.1/tcp/8082
!nohup ipfs daemon > startup.log &
```

    --2022-09-30 19:10:07--  https://dist.ipfs.io/go-ipfs/v0.4.2/go-ipfs_v0.4.2_linux-amd64.tar.gz
    Resolving dist.ipfs.io (dist.ipfs.io)... 209.94.78.1, 2602:fea2:3::1
    Connecting to dist.ipfs.io (dist.ipfs.io)|209.94.78.1|:443... connected.
    HTTP request sent, awaiting response... 200 OK
    Length: 7642422 (7.3M) [application/gzip]
    Saving to: ‘go-ipfs_v0.4.2_linux-amd64.tar.gz.1’
    
    go-ipfs_v0.4.2_linu 100%[===================>]   7.29M  40.8MB/s    in 0.2s    
    
    2022-09-30 19:10:07 (40.8 MB/s) - ‘go-ipfs_v0.4.2_linux-amd64.tar.gz.1’ saved [7642422/7642422]
    
    go-ipfs/build-log
    go-ipfs/install.sh
    go-ipfs/ipfs
    go-ipfs/LICENSE
    go-ipfs/README.md
    initializing ipfs node at /root/.ipfs
    Error: ipfs configuration file already exists!
    Reinitializing would overwrite your keys.
    
    Hello and Welcome to IPFS!
    
    ██╗██████╗ ███████╗███████╗
    ██║██╔══██╗██╔════╝██╔════╝
    ██║██████╔╝█████╗  ███████╗
    ██║██╔═══╝ ██╔══╝  ╚════██║
    ██║██║     ██║     ███████║
    ╚═╝╚═╝     ╚═╝     ╚══════╝
    
    If you're seeing this, you have successfully installed
    IPFS and are now interfacing with the ipfs merkledag!
    
     -------------------------------------------------------
    | Warning:                                              |
    |   This is alpha software. Use at your own discretion! |
    |   Much is missing or lacking polish. There are bugs.  |
    |   Not yet secure. Read the security notes for more.   |
     -------------------------------------------------------
    
    Check out some of the other files in this directory:
    
      ./about
      ./help
      ./quick-start     <-- usage examples
      ./readme          <-- this file
      ./security-notes
    nohup: redirecting stderr to stdout



```python
!cd data
!ipfs get QmdnMxSSvD8QYR6F4S7wkgQsW16bR6U7zyDTbiEm72RPpB
```


**Running the command on bacalhau**

The command can be broken down into 4 pieces

`bacalhau docker run` using the docker backend

`-v QmaNyzSpJCt1gMCQLd3QugihY6HzdYmA8QMEa45LDBbVPz:/data` here we mount the CID of the dataset we uploaded to IPFS and mount it to a folder called data on the container

`nipreps/mriqc:latest` the name and the tag of the docker image we are using


```
mriqc ../data/ds005 ../outputs participant --participant_label 01 02 03
```


This is the command that we run where we specify path to the `../data/ds005` input dataset

`../outputs` path where we want to save our outputs,

`participant --participant_label 01 02 03` Run the participant level in subjects 001 002 003


```
bacalhau docker run \
-v QmaNyzSpJCt1gMCQLd3QugihY6HzdYmA8QMEa45LDBbVPz:/data \
nipreps/mriqc:latest \
-- mriqc ../data/ds005 ../outputs participant --participant_label 01 02 03
```


Insalling bacalhau


```python
!curl -sL https://get.bacalhau.org/install.sh | bash
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
-v QmaNyzSpJCt1gMCQLd3QugihY6HzdYmA8QMEa45LDBbVPz:/data \
nipreps/mriqc:latest 
-- mriqc ../data/ds005 ../outputs participant --participant_label 01 02 03
```


```python
%env JOB_ID={job_id}
```


Running the commands will output a UUID (like `54506541-4eb9-45f4-a0b1-ea0aecd34b3e`). This is the ID of the job that was created. You can check the status of the job with the following command:



```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
```


Where it says "`Completed`", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```bash
%%bash
bacalhau describe ${JOB_ID}
```

To Download the results of your job, run 

---

the following command:


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



The structure of the files and directories will look like this:


```
.
├── shards
│   └── job-8e89eb2f-1ae7-4b92-ba72-8abfade02a23-shard-0-host-QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
│       ├── exitCode
│       ├── stderr
│       └── stdout
├── stderr
├── stdout
└── volumes
    └── outputs
        ├── dataset_description.json
        ├── sub-01_T1w.html
        ├── sub-01_T1w.json
        ├── sub-01_task-mixedgamblestask_run-01_bold.html
        ├── sub-01_task-mixedgamblestask_run-01_bold.json
        ├── sub-01_task-mixedgamblestask_run-02_bold.html
        ├── sub-01_task-mixedgamblestask_run-02_bold.json
        ├── sub-01_task-mixedgamblestask_run-03_bold.html
        ├── sub-01_task-mixedgamblestask_run-03_bold.json
        ├── sub-02_T1w.html
        ├── sub-02_T1w.json
        ├── sub-02_task-mixedgamblestask_run-01_bold.html
        ├── sub-02_task-mixedgamblestask_run-01_bold.json
        ├── sub-02_task-mixedgamblestask_run-02_bold.html
        ├── sub-02_task-mixedgamblestask_run-02_bold.json
        ├── sub-02_task-mixedgamblestask_run-03_bold.html
        ├── sub-02_task-mixedgamblestask_run-03_bold.json
        ├── sub-03_T1w.html
        ├── sub-03_T1w.json
        ├── sub-03_task-mixedgamblestask_run-01_bold.html
        ├── sub-03_task-mixedgamblestask_run-01_bold.json
        ├── sub-03_task-mixedgamblestask_run-02_bold.html
        ├── sub-03_task-mixedgamblestask_run-02_bold.json
        ├── sub-03_task-mixedgamblestask_run-03_bold.html
        └── sub-03_task-mixedgamblestask_run-03_bold.json
```



    The outputs of your job is in volumes/outputs



* Volumes folder contains the outputs of our job
* stdout contains things printed to the console like outputs, etc.
* stderr contains any errors. In this case, since there are no errors, it's will be empty
