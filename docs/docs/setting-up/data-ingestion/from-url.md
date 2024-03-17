---
sidebar_label: "From A URL"
sidebar_position: 1
---
# Copy Data from a URL to a Public Storage

To upload a file from a URL we will use the `bacalhau docker run` command.

```bash
%%bash --out job_id
bacalhau docker run \
    --id-only \
    --wait \
    --input https://raw.githubusercontent.com/filecoin-project/bacalhau/main/README.md \
    ghcr.io/bacalhau-project/examples/upload:v1
```

The job has been submitted and Bacalhau has printed out the related job id. We store that in an environment variable so that we can reuse it later on.

### Structure of the command

Let's look closely at the command above:

* `bacalhau docker run`: call to bacalhau

* `ghcr.io/bacalhau-project/examples/upload:v1`: the name and the tag of the docker image we are using

* ` --input=https://raw.githubusercontent.com/filecoin-project/bacalhau/main/README.md \`: URL path of the input data volumes downloaded from a URL source.

The `bacalhau docker run` command takes advantage of the `--input` parameter. This will download a file from a public URL and place it in the `/inputs` directory of the container (by default). Then we will use a helper container to move that data to the `/outputs` directory so that it is published to your public storage via IPFS. In our case we are using Filecoin as our public storage.

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

```bash
%%bash
bacalhau describe  $JOB_ID
```

**Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.

```bash
%%bash
rm -rf results && mkdir ./results
bacalhau get --output-dir ./results $JOB_ID
```

## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**, **per_shard files**, and the **raw** directory. To view the file, run the following command:

```bash
%%bash
head -n 15 ./results/combined_results/stdout
```

## Get the CID From the Completed Job

To get the output CID from a completed job, run the following command:

```bash
%%bash --out cid
bacalhau list $JOB_ID --output=json | jq -r '.[0].Status.JobState.Nodes[] | .Shards."0".PublishedResults | select(.CID) | .CID'
```
The job will upload the CID to your public storage via IPFS. We will store the _cid_ that in an environment variable so that we can reuse it later on.

### Use the CID in a New Bacalhau Job

Now that we have the CID, we can use it in a new job. This time we will use the `--input` parameter to tell Bacalhau to use the CID we just uploaded.

In this case, our "job" is just to list the contents of the `/inputs` directory. You can see that the "input" data is located under `/inputs/outputs/README.md`.


```bash
%%bash --out job_id
bacalhau docker run \
    --id-only \
    --wait \
    --input ipfs://$CID \
    ubuntu -- \
    bash -c "set -x; ls -l /inputs; ls -l /inputs/outputs; cat /inputs/outputs/README.md"
```

The job has been submitted and Bacalhau has printed out the related job id. We store that in an environment variable so that we can reuse it later on.


## Need Support?

For questions and feedback, please reach out in our [forum](https://github.com/filecoin-project/bacalhau/discussions)
