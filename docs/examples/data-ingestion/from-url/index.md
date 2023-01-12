---
sidebar_label: "From A URL"
sidebar_position: 1
---
# Copy Data from a URL to Filecoin

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-ingestion/from-url/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-ingestion/from-url/index.ipynb)


The goal of the Bacalhau project is to make it easy to perform distributed, decentralised computation next to where the data resides. So a key step in this process is making your data accessible.

IPFS is a set of protocols that allow data to be discovered and accessed in a decentralised way. Data is identified by its content identifier (CID) and can be accessed by anyone who knows the CID.

This notebook shows how to use Bacalhau to copy data from a URL to Filecoin and expose on IPFS for use with Bacalhau.

This takes advantage of the fact that all Bacalhau jobs are published to a Filecoin contract via Estuary. All data that is located in the `/outputs` directory is published.

The example below uses a simple tool we have created to help make it easier to move data in Bacalhau.

### Prerequisites

* [The Bacalhau client](https://docs.bacalhau.org/getting-started/installation)
* [`jq` to parse the Bacalhau output](https://stedolan.github.io/jq/download/)

## 1. Uploading A File From a URL

To upload a file from a URL we will take advantage of the `--input-urls` parameter of the `bacalhau docker run` command. This will download a file from a public URL and place it in the `/inputs` directory of the container (by default).

Then we will use a helper container to move that data to the `/outputs` directory so that it is published to Filecoin via Estuary.

:::tip
You can find out more about the [helper container in the examples repository](https://github.com/bacalhau-project/examples/tree/main/tools/upload).
:::


```bash
bacalhau docker run \
    --id-only \
    --wait \
    --input-urls=https://raw.githubusercontent.com/filecoin-project/bacalhau/main/README.md \
    ghcr.io/bacalhau-project/examples/upload:v1
```

    env: JOB_ID=418f5335-8023-42ca-b65f-7844614151f0


Just to be safe, double check that the job succeeded by running the describe command (and some `jq` to parse it).


```bash
bacalhau list $JOB_ID --output=json | jq '.[0].Status.JobState.Nodes[] | .Shards."0" | select(.RunOutput)'
```

    {
      "NodeId": "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "State": "Completed",
      "VerificationResult": {
        "Complete": true,
        "Result": true
      },
      "PublishedResults": {
        "StorageSource": "IPFS",
        "Name": "job-418f5335-8023-42ca-b65f-7844614151f0-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
        "CID": "QmYT1RuLmhqh6xdXLG62kLjn2G513nHiWmuy6j6vm5QT5H"
      },
      "RunOutput": {
        "stdout": "1:45PM INF Copying files InputPath=/inputs OutputPath=/outputs\n1:45PM INF Copying object dst=/outputs/README.md src=/inputs/README.md\n1:45PM INF Done copying all objects files=[\"/outputs\",\"/outputs/README.md\"]\n",
        "stdouttruncated": false,
        "stderr": "",
        "stderrtruncated": false,
        "exitCode": 0,
        "runnerError": ""
      }
    }


## 2. Get the CID From the Completed Job

The job will upload the CID to the Filecoin network via Estuary. Let's get the CID from the output.


```bash
bacalhau list $JOB_ID --output=json | jq -r '.[0].Status.JobState.Nodes[] | .Shards."0".PublishedResults | select(.CID) | .CID'
```

    env: CID=QmYT1RuLmhqh6xdXLG62kLjn2G513nHiWmuy6j6vm5QT5H



Your CID is <b>QmYT1RuLmhqh6xdXLG62kLjn2G513nHiWmuy6j6vm5QT5H
.</b><br/><br/><a href="https://ipfs.io/ipfs/QmYT1RuLmhqh6xdXLG62kLjn2G513nHiWmuy6j6vm5QT5H
"><button>View files on ipfs.io</button></a>


## 3. Use the CID in a New Bacalhau Job

Now that we have the CID, we can use it in a new job. This time we will use the `--inputs` parameter to tell Bacalhau to use the CID we just uploaded.

In this case my "job" is just to list the contents of the `/inputs` directory and cat the file we downloaded in the first step. You can see that the "input" data is located under `/inputs/outputs/README.md`.


```bash
bacalhau docker run \
    --id-only \
    --wait \
    --inputs=$CID \
    ubuntu -- \
    bash -c "set -x; ls -l /inputs; ls -l /inputs/outputs; cat /inputs/outputs/README.md"
```

    env: JOB_ID=37e3c424-072a-4ea5-bc3a-76909dce17ee



```bash
rm -rf results && mkdir ./results
bacalhau get --output-dir ./results $JOB_ID 
```

    Fetching results of job '37e3c424-072a-4ea5-bc3a-76909dce17ee'...
    Results for job '37e3c424-072a-4ea5-bc3a-76909dce17ee' have been written to...
    ./results


    2023/01/12 13:45:45 CleanupManager.fnsMutex violation CRITICAL section took 22.714ms 22714000 (threshold 10ms)



```python
%cat ./results/combined_results/stdout
```

    total 12
    -rw-r--r-- 1 root root    1 Jan 12 13:45 exitCode
    drwxr-xr-x 2 root root 4096 Jan 12 13:45 outputs
    -rw-r--r-- 1 root root    0 Jan 12 13:45 stderr
    -rw-r--r-- 1 root root  210 Jan 12 13:45 stdout
    total 4
    -rw-r--r-- 1 root root 3802 Jan 12 13:45 README.md
    <!-- commenting out until we can fix the image logo [![CircleCI](https://dl.circleci.com/status-badge/img/null/filecoin-project/bacalhau/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/null/filecoin-project/bacalhau/tree/main)
    -->
    # The Filecoin Distributed Computation Framework  
    <p align="center">
      <img src="docs/images/bacalhau-fish.jpg" alt="Bacalhau Logo" width="400" />
    </p>
    <p align=center>
      Compute Over Data == CoD
      <br>
      Bacalhau == "Salted CoD Fish" (Portuguese)
    </p>
      
    <br>
    
    The purpose of Bacalhau is to provide a platform for public, transparent, and optionally verifiable computation. Bacalhau enables users to run arbitrary docker containers and wasm images as tasks against data stored in IPFS. This architecture is also referred to as Compute Over Data (or CoD). The Portuguese word for salted Cod fish is "Bacalhau" which is the origin of the project's name.
    
    Initially, the Bacalhau project will focus on serving data processing and analytics use cases. Over time Bacalhau will expand to other compute workloads, learn more about it future plans in the [roadmap document](ROADMAP.md).
    
    * [Getting Started](https://docs.bacalhau.org/getting-started/installation) ⚡
    * [Documentation](https://docs.bacalhau.org/) :closed_book:
    * [Slack Community](https://filecoin.io/slack) is open to anyone! Join the `#bacalhau` channel :raising_hand:
    * [Code Examples Repository](https://github.com/bacalhau-project/examples) :mag:
    
    Watch a 90 seconds demo of Bacalhau in action:
    
    <p align=center>
      <a href="https://www.youtube.com/watch?v=4YHkmL4Ld74" target="_blank">
        <img src="https://github.com/filecoin-project/bacalhau/raw/a49f4e9c89acce2890aa444fdbb5aa47674ede68/docs/images/thumb-bacalhau-demo-1st-july.jpg" alt="Watch the video" width="580" border="10" />
      </a>
    </p>
    
    
    Learn more about the project from our [Website](https://www.bacalhau.org/), [Twitter](https://twitter.com/BacalhauProject) & [YouTube Channel](https://www.youtube.com/channel/UC45IQagLzNR3wdNCUn4vi0A).
    
    ## Latest Updates
    
    * [Weekly Bacalhau Project Reports](https://github.com/filecoin-project/bacalhau/wiki)
    * [Bacalhau Overview at DeSci Berlin June 2022](https://www.youtube.com/watch?v=HA8ijt4dzAY)
    
    
    ## Getting Started
    
    Please see the instructions here to get started running a hello example: [Getting Started with Bacalhau](https://docs.bacalhau.org/getting-started/installation).
    For a more data intensive demo, check out the [Image Processing tutorial](https://docs.bacalhau.org/examples/data-engineering/image-processing/).
    
    ## Getting Help
    
    For usage questions or issues reach out the Bacalhau team either in the [Slack channel](https://filecoinproject.slack.com/archives/C02RLM3JHUY) or open a new issue here on github.
    
    ## Developer Guide
    
    ### Running Bacalhau locally
    
    Developers can spin up bacalhau and run a local demo using the `devstack` command. 
    Please see [docs/running_locally.md](docs/running_locally.md) for instructions.
    Also, see [docs/debugging_locally.md](docs/debugging_locally.md) for some useful tricks for debugging.
    
    ### Release a new version
    
    To ship a new version of the CLI & Bacalhau network please follow the instuctions at [docs/create_release.md](docs/create_release.md).
    
    ### Notes for Contributors
    
    Bacalhau's CI pipeline performs a variety of linting and formatting checks on new pull requests. 
    To have these checks run locally when you make a new commit, you can use the precommit hook in `./githooks`:
    
    ```bash
    git config core.hooksPath ./githooks
    ```
    
    If you want to run the linter manually:
    
    ```bash
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/local/go/bin
    golangci-lint --version
    make lint
    ```
    
    The config lives in `.golangci.yml`
    
    ## Licence
    
    [Apache-2.0](./LICENSE)

