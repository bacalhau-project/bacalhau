---
sidebar_label: "From A URL"
sidebar_position: 1
---
# Copy Data from a URL to a Public Storage

To upload a file from a URL we will use the `bacalhau docker run` command.

```bash
bacalhau docker run \
    --id-only \
    --wait \
    --input https://raw.githubusercontent.com/filecoin-project/bacalhau/main/README.md \
    ghcr.io/bacalhau-project/examples/upload:v1
```

The job has been submitted and Bacalhau has printed out the related job id.

### Structure of the command

Let's look closely at the command above:

<<<<<<< HEAD
* `bacalhau docker run`: call to bacalhau using docker executor
=======
* `bacalhau docker run`: call to bacalhau
>>>>>>> main

* `--input https://raw.githubusercontent.com/filecoin-project/bacalhau/main/README.md`: URL path of the input data volumes downloaded from a URL source. 

<<<<<<< HEAD
* `ghcr.io/bacalhau-project/examples/upload:v1`: the name and tag of the docker image we are using

The `bacalhau docker run` command takes advantage of the `--input` parameter. This will download a file from a public URL and place it in the `/inputs` directory of the container (by default). Then we will use a helper container to move that data to the /outputs directory.
=======
* ` --input=https://raw.githubusercontent.com/filecoin-project/bacalhau/main/README.md \`: URL path of the input data volumes downloaded from a URL source.
>>>>>>> main


:::tip
You can find out more about the [helper container in the examples repository](https://github.com/bacalhau-project/examples/tree/main/tools/upload) which is designed to simplify the data uploading process.
:::
:::info
For more details, see the [CLI commands guide](../../dev/cli-reference/all-flags.md#docker-run)
:::

## Checking the State of Your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`, processing the json ouput with the `jq`:

```bash
bacalhau list $JOB_ID --output=json | jq '.[0].Status.JobState.Nodes[] | .Shards."0" | select(.RunOutput)'
```

When the job status is `Published` or `Completed`, that means the job is done, and we can get the results using the job ID.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.

```bash
<<<<<<< HEAD
bacalhau describe  $JOB_ID 
=======
%%bash
bacalhau describe  $JOB_ID
>>>>>>> main
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we removed a directory in case it was present before, created it and downloaded our job output to be stored in that directory.

```bash
rm -rf results && mkdir ./results
bacalhau get --output-dir ./results $JOB_ID
```

## Viewing your Job Output

Each job result contains an `outputs` subfolder and `exitCode`, `stderr` and `stdout` files with relevant content. To view the execution logs execute following:

```bash
head -n 15 ./results/stdout
```
And to view the job execution result (`README.md` file in the example case), which was saved as a job output, execute:
```bash
tail ./results/outputs/README.md
```


## Get the CID From the Completed Job

To get the output CID from a completed job, run the following command:

```bash
bacalhau list $JOB_ID --output=json | jq -r '.[0].Status.JobState.Nodes[] | .Shards."0".PublishedResults | select(.CID) | .CID'
```

The job will upload the CID to the public storage via IPFS. We will store the CID in an environment variable so that we can reuse it later on.


## Use the CID in a New Bacalhau Job

Now that we have the CID, we can use it in a new job. This time we will use the `--input` parameter to tell Bacalhau to use the CID we just uploaded.

In this case, the only goal of our job is just to list the contents of the `/inputs` directory. You can see that the "input" data is located under `/inputs/outputs/README.md`.


```bash
bacalhau docker run \
    --id-only \
    --wait \
    --input ipfs://$CID \
    ubuntu -- \
    bash -c "set -x; ls -l /inputs; ls -l /inputs/outputs; cat /inputs/outputs/README.md"
```

The job has been submitted and Bacalhau has printed out the related job id. We store that in an environment variable so that we can reuse it later on.


## Need Support?

For questions and feedback, please reach out in our [Slack](https://bacalhauproject.slack.com)
