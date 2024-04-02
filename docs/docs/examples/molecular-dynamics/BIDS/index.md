---
sidebar_label: BIDS
sidebar_position: 1
---
# Running BIDS Apps on Bacalhau


In this example tutorial, we will look at how to run BIDS App on Bacalhau. BIDS (Brain Imaging Data Structure) is an emerging standard for organizing and describing neuroimaging datasets.  [BIDS App](https://bids-apps.neuroimaging.io/about/) is a container image capturing a neuroimaging pipeline that takes a BIDS formatted dataset as input. Each BIDS App has the same core set of command line arguments, making them easy to run and integrate into automated platforms. BIDS Apps are constructed in a way that does not depend on any software outside of the image other than the container engine.


## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)

## Downloading datasets

For this tutorial, download file `ds005.tar` from this Bids dataset [folder](https://drive.google.com/drive/folders/0B2JWN60ZLkgkMGlUY3B4MXZIZW8?resourcekey=0-EYVSOlRbxeFKO8NpjWWM3w) and untar it in a directory: 

```bash
%%bash
mkdir data
tar -xf ds005.tar -C data 
```

Let's take a look at the structure of the `data` directory:


```
data
└── ds005
    ├── CHANGES
    ├── dataset_description.json
    ├── participants.tsv
    ├── README
    ├── sub-01
    │   ├── anat
    │   │   ├── sub-01_inplaneT2.nii.gz
    │   │   └── sub-01_T1w.nii.gz
    │   └── func
    │       ├── sub-01_task-mixedgamblestask_run-01_bold.nii.gz
    │       ├── sub-01_task-mixedgamblestask_run-01_events.tsv
    │       ├── sub-01_task-mixedgamblestask_run-02_bold.nii.gz
    │       ├── sub-01_task-mixedgamblestask_run-02_events.tsv
    │       ├── sub-01_task-mixedgamblestask_run-03_bold.nii.gz
    │       └── sub-01_task-mixedgamblestask_run-03_events.tsv
    ├── sub-02
    │   ├── anat
    │   │   ├── sub-02_inplaneT2.nii.gz
    │   │   └── sub-02_T1w.nii.gz
    ...
```

## Uploading the datasets to IPFS

The simplest way to upload the data to IPFS is to use a third-party service to "pin" data to the IPFS network, to ensure that the data exists and is available. To do this, you need an account with a pinning service like [Pinata](https://app.pinata.cloud/pinmanager) or [nft.storage](https://nft.storage/docs/how-to/nftup/). Once registered, you can use their UI or API or SDKs to upload files.

When you pin your data, you'll get a CID which is in a format like this `QmaNyzSpJCt1gMCQLd3QugihY6HzdYmA8QMEa45LDBbVPz`. Copy the CID as it will be used to access your data


:::info
Alternatively, you can upload your dataset to IPFS using [IPFS CLI](https://docs.ipfs.tech/install/command-line/#official-distributions), but the recommended approach is to use a pinning service as we have mentioned above.
:::

## Running a Bacalhau Job


```bash
%%bash --out job_id
bacalhau docker run \
    --id-only \
    --wait \
    --timeout 3600 \
    --wait-timeout-secs 3600 \
    -i ipfs://QmaNyzSpJCt1gMCQLd3QugihY6HzdYmA8QMEa45LDBbVPz:/data \
    nipreps/mriqc:latest \
    -- mriqc ../data/ds005 ../outputs participant --participant_label 01 02 03
```

### Structure of the command

Let's look closely at the command above:

`bacalhau docker run`: call to bacalhau

`-i ipfs://QmaNyzSpJCt1gMCQLd3QugihY6HzdYmA8QMEa45LDBbVPz:/data`: mount the CID of the dataset that is uploaded to IPFS and mount it to a folder called data on the container

`nipreps/mriqc:latest`: the name and the tag of the docker image we are using

`../data/ds005`: path to input dataset

`../outputs`: path to the output

`participant --participant_label 01 02 03`: run the mriqc on subjects with participant labels 01, 02, and 03


When a job is submitted, Bacalhau prints out the related job_id. We store that in an environment variable so that we can reuse it later on.


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
ls results/ # list the contents of the current directory 
cat results/stdout # displays the contents of the current directory 
```

## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).