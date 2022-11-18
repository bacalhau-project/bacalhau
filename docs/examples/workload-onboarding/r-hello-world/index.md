---
sidebar_label: "R - Hello World"
sidebar_position: 50
---
# Running a simple R script


# Introduction

You can choose the official standard containers for each language like R

and run them on bacalhau, please make sure that you have all the dependencies installed and the scripts shouldn’t make any external requests or install dependencies as networking is disabled in bacalhau


## **Running Locally**

To install R follow these instructions [A Installing R and RStudio | Hands-On Programming with R](https://rstudio-education.github.io/hopr/starting.html) 

After R and RStudio is installed

Create a Script called hello.R

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/r-hello-world/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/r-hello-world/index.ipynb)


```python
%%writefile hello.R
print("hello world")
```

    Overwriting hello.R



the print() function is used to print output in R

Run the script



```bash
Rscript hello.R
```

    [1] "hello world"


Install and start IPFS


```bash
wget https://dist.ipfs.io/go-ipfs/v0.4.2/go-ipfs_v0.4.2_linux-amd64.tar.gz
tar xvfz go-ipfs_v0.4.2_linux-amd64.tar.gz
mv go-ipfs/ipfs /usr/local/bin/ipfs
ipfs init
ipfs cat /ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme
ipfs config Addresses.Gateway /ip4/127.0.0.1/tcp/8082
ipfs config Addresses.API /ip4/127.0.0.1/tcp/5002
nohup ipfs daemon > startup.log &
```



If the script runs successfully, Add the hello.R script to IPFS, using the ipfs add command



```bash
ipfs add hello.R
```

    
    21 B / 21 B  100.00 % 0[2K
    added QmVHSWhAL7fNkRiHfoEJGeMYjaYZUsKHvix7L54SptR8ie hello.R




## **Running on bacalhau**

To run the script we are using r-base as a container And mounting the Uploaded CID to it

Command:


```
bacalhau docker run \
 -v QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk:/hello.R \
 r-base \
-- Rscript hello.R
```


Insalling bacalhau


```bash
curl -sL https://get.bacalhau.org/install.sh | bash
```

    Your system is linux_amd64
    
    BACALHAU CLI is detected:
    Client Version: v0.2.5
    Server Version: v0.2.5
    Reinstalling BACALHAU CLI - /usr/local/bin/bacalhau...
    Getting the latest BACALHAU CLI...
    Installing v0.2.5 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.2.5/bacalhau_v0.2.5_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.2.5/bacalhau_v0.2.5_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into /usr/local/bin successfully.
    Client Version: v0.2.5
    Server Version: v0.2.5



```bash
bacalhau docker run \
--wait \
--wait-timeout-secs 1000 \
--id-only \
-v QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk:/hello.R \
r-base \
-- Rscript hello.R
```

    c1334838-d75e-413c-b5d1-2a8cf3a0e847



```python
%env JOB_ID={job_id}
```


Running the commands will output a UUID (like `54506541-4eb9-45f4-a0b1-ea0aecd34b3e`). This is the ID of the job that was created. You can check the status of the job with the following command:



```bash
bacalhau list --id-filter ${JOB_ID}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 17:53:19 [0m[97;40m c1334838 [0m[97;40m Docker r-base Rscrip... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmQ1Yci2Gbptoc... [0m



Where it says "`Published `", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```bash
bacalhau describe  ${JOB_ID}
```

    JobAPIVersion: ""
    ID: c1334838-d75e-413c-b5d1-2a8cf3a0e847
    RequesterNodeID: QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3
    ClientID: 2f3ace1e78ecef12af7b0547496393f45508eb8ab9c5c71dbcd56a867dab46cb
    Spec:
        Engine: 2
        Verifier: 1
        Publisher: 4
        Docker:
            Image: r-base
            Entrypoint:
                - Rscript
                - hello.R
        inputs:
            - Engine: 1
              Cid: QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk
              path: /hello.R
        outputs:
            - Engine: 1
              Name: outputs
              path: /outputs
        Sharding:
            BatchSize: 1
            GlobPatternBasePath: /inputs
    Deal:
        Concurrency: 1
    CreatedAt: 2022-10-01T17:53:19.581955821Z
    JobState:
        Nodes:
            QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF:
                Shards:
                    0:
                        NodeId: QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF
                        ShardIndex: 0
                        State: 7
                        Status: 'Got results proposal of length: 0'
                        VerificationProposal: []
                        VerificationResult:
                            Complete: true
                            Result: true
                        PublishedResults:
                            Engine: 1
                            Name: job-c1334838-d75e-413c-b5d1-2a8cf3a0e847-shard-0-host-QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF
                            Cid: QmQ1Yci2GbptoccHy46txBK28gtnbKcb7nvFNHYpS6Gbn7
                        RunOutput:
                            Stdout: '[1] "hello world"'
                            StdoutTruncated: false
                            Stderr: ""
                            StderrTruncated: false
                            ExitCode: 0
                            RunnerError: ""


Since there is no error we can’t see any error instead we see the state of our job to be complete, that means 
we can download the results!
we create a temporary directory to save our results


```bash
mkdir results
```

    mkdir: cannot create directory ‘results’: File exists


To Download the results of your job, run 

---

the following command:


```bash
bacalhau get ${JOB_ID} --output-dir results
```

    [90m17:53:36.606 |[0m [32mINF[0m [1mbacalhau/get.go:67[0m[36m >[0m Fetching results of job 'c1334838-d75e-413c-b5d1-2a8cf3a0e847'...
    2022/10/01 17:53:36 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.
    [90m17:53:46.792 |[0m [32mINF[0m [1mipfs/downloader.go:115[0m[36m >[0m Found 1 result shards, downloading to temporary folder.
    [90m17:53:48.178 |[0m [32mINF[0m [1mipfs/downloader.go:195[0m[36m >[0m Combining shard from output volume 'outputs' to final location: '/content/results'


After the download has finished you should 
see the following contents in results directory


```bash
ls results/
```

    shards	stderr	stdout	volumes


Viewing the result


```bash
cat results/combined_results/stdout
```

    [1] "hello world"




Mounting the script from a URL in this case a github gist

Command:


```
bacalhau docker run \
-u https://gist.github.com/js-ts/7a865dda1e1f968e4de86fcc4e710dad:/hello.R \
r-base \
-- Rscript hello.R
```




```bash
bacalhau describe ${JOB_ID} --spec > job.yaml
```


```bash
cat job.yaml
```
