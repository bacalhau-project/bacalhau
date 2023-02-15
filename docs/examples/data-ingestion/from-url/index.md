---
sidebar_label: "From A URL"
sidebar_position: 1
---
# Copy Data from a URL to Filecoin

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-ingestion/from-url/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-ingestion/from-url/index.ipynb)


In this example tutorial, we will show how to use Bacalhau to copy data from a URL to Filecoin and expose on IPFS for use with Bacalhau. IPFS is a set of protocols that allow data to be discovered and accessed in a decentralised way. Data is identified by its content identifier (CID) and can be accessed by anyone who knows the CID.

The goal of the Bacalhau project is to make it easy to perform distributed, decentralised computation next to where the data resides. So a key step in this process is making your data accessible.



## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Running a Bacalhau Job 

To upload a file from a URL we will use the `bacalhau docker run`.


```bash
%%bash --out job_id
bacalhau docker run \
    --id-only \
    --wait \
    --input-urls=https://raw.githubusercontent.com/filecoin-project/bacalhau/main/README.md \
    ghcr.io/bacalhau-project/examples/upload:v1
```

The job has been submitted and Bacalhau has printed out the related job id. We store that in an environment variable so that we can reuse it later on.

The `bacalhau docker run` command takes advantage of the `--input-urls` parameter. This will download a file from a public URL and place it in the `/inputs` directory of the container (by default). Then we will use a helper container to move that data to the `/outputs` directory so that it is published to Filecoin via Estuary.

:::tip
You can find out more about the [helper container in the examples repository](https://github.com/bacalhau-project/examples/tree/main/tools/upload).
:::

## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
bacalhau list $JOB_ID --output=json | jq '.[0].Status.JobState.Nodes[] | .Shards."0" | select(.RunOutput)'
```

When it says `Published` or `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


All Bacalhau jobs are published to a Filecoin contract via Estuary. All data that is located in the `/outputs` directory is published.


### Get the CID From the Completed Job

The job will upload the CID to the Filecoin network via Estuary. Let's get the CID from the output.


```bash
%%bash --out cid
bacalhau list $JOB_ID --output=json | jq -r '.[0].Status.JobState.Nodes[] | .Shards."0".PublishedResults | select(.CID) | .CID'
```

We will store the _cid_ that in an environment variable so that we can reuse it later on.

    env: CID=QmYT1RuLmhqh6xdXLG62kLjn2G513nHiWmuy6j6vm5QT5H


### Use the CID in a New Bacalhau Job

Now that we have the CID, we can use it in a new job. This time we will use the `--inputs` parameter to tell Bacalhau to use the CID we just uploaded.

In this case, our "job" is just to list the contents of the `/inputs` directory. You can see that the "input" data is located under `/inputs/outputs/README.md`.


```bash
%%bash --out job_id
bacalhau docker run \
    --id-only \
    --wait \
    --inputs=$CID \
    ubuntu -- \
    bash -c "set -x; ls -l /inputs; ls -l /inputs/outputs; cat /inputs/outputs/README.md"
```

The job has been submitted and Bacalhau has printed out the related job id. We store that in an environment variable so that we can reuse it later on.

    env: JOB_ID=37e3c424-072a-4ea5-bc3a-76909dce17ee


**Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir ./results
bacalhau get --output-dir ./results $JOB_ID 
```

    Fetching results of job '37e3c424-072a-4ea5-bc3a-76909dce17ee'...
    Results for job '37e3c424-072a-4ea5-bc3a-76909dce17ee' have been written to...
    ./results


    2023/01/12 13:45:45 CleanupManager.fnsMutex violation CRITICAL section took 22.714ms 22714000 (threshold 10ms)


## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**, **per_shard files**, and the **raw** directory. To view the file, run the following command:


```bash
%%bash
head -n 15 ./results/combined_results/stdout
```

## Need Support?

For questions, feedback, please reach out in our [forum](https://github.com/filecoin-project/bacalhau/discussions)
