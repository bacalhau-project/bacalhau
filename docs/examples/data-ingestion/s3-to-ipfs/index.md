---
sidebar_label: "From S3"
sidebar_position: 3
---
# Copy Data from S3 to IPFS


[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-ingestion/s3-to-ipfs/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-ingestion/s3-to-ipfs/index.ipynb)


In this tutorial, to copy Data from S3 to IPFS, we will scrape all the links from a public AWS S3 buckets and then copy the data to IPFS using Bacalhau. 


## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Running a Bacalhau Job

If your bucket has more than 1000 files, with the command below, you can submit a Bacalhau job to extract the URL list of the files.


```bash
%%bash --out job_id
bacalhau docker run \
-u https://noaa-goes16.s3.amazonaws.com/ \
-v QmR1qXs8Y8T7G6F2Yy91sDTWG6WAhoFrCjMGRvy7N1y5LC:/extract.py \
--id-only \
--wait \
python \
-- /bin/bash -c 'python3 extract.py https://noaa-goes16.s3.amazonaws.com/  /inputs'
```


Before running the command above, replace the following:

- `-u  https://noaa-goes16.s3.amazonaws.com/`: we replace the placeholders with `noaa-goes16` which is the name of the bucket we want to extract URLs from

-  `-v QmR1qXs8Y8T7G6F2Yy91sDTWG6WAhoFrCjMGRvy7N1y5LC:/extract.py \`: Mounting the scrapper script, this script extracts the links from the XML document tree

 - `-- /bin/bash -c 'python3 extract.py https://noaa-goes16.s3.amazonaws.com/  /inputs'`: Executing the scrapper script

The command above extracts the path of the file in the bucket, we added the URL as a prefix to the path `https://noaa-goes16.s3.amazonaws.com/`  then provided the path where the XML document tree of the URL is mounted which is `/inputs`

When a job is sumbitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

:::tip
There are certain limitations to this step, as this only works with datasets that are publicly accessible and don't require an AWS account or pay to use buckets and possibly only limited to first 1000 URLs.
:::

## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`. 


```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
```

    [92;100m CREATED           [0m[92;100m ID                                   [0m[92;100m JOB                                                                                          [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED                                            [0m
    [97;40m 22-11-13-13:52:12 [0m[97;40m 12e1b4d9-00b0-4824-bbd1-6d75083dcae0 [0m[97;40m Docker python /bin/bash -c python3 extract.py https://noaa-goes16.s3.amazonaws.com/  /inputs [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmaxiCCJ5vuwEfA2x7VVvMUXHxHN6iYNPhmvFhXSyUyNYx [0m


When it says `Published` or `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe ${JOB_ID}
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir -p results # Temporary directory to store the results
bacalhau get $JOB_ID --output-dir results # Download the results
```

    Fetching results of job '12e1b4d9-00b0-4824-bbd1-6d75083dcae0'...
    Results for job '12e1b4d9-00b0-4824-bbd1-6d75083dcae0' have been written to...
    results


    2022/11/13 13:53:09 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.


After the download has finished you should see the following contents in results directory.

## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**, **per_shard files**, and the **raw** directory. To view your file, run the following command:


```bash
%%bash
head -10 results/combined_results/stdout
```

    https://noaa-goes16.s3.amazonaws.com/ABI-L1b-RadC/2000/001/12/OR_ABI-L1b-RadC-M3C01_G16_s20000011200000_e20000011200000_c20170671748180.nc
    https://noaa-goes16.s3.amazonaws.com/ABI-L1b-RadC/2000/001/12/OR_ABI-L1b-RadC-M3C01_G16_s20000011200000_e20000011200000_c20170691603180.nc
    https://noaa-goes16.s3.amazonaws.com/ABI-L1b-RadC/2000/001/12/OR_ABI-L1b-RadC-M3C01_G16_s20000011200000_e20000011200000_c20170751219598.nc
    https://noaa-goes16.s3.amazonaws.com/ABI-L1b-RadC/2000/001/12/OR_ABI-L1b-RadC-M3C01_G16_s20000011200000_e20000011200000_c20170752149454.nc
    https://noaa-goes16.s3.amazonaws.com/ABI-L1b-RadC/2000/001/12/OR_ABI-L1b-RadC-M3C01_G16_s20000011200000_e20000011200000_c20170752204183.nc
    https://noaa-goes16.s3.amazonaws.com/ABI-L1b-RadC/2000/001/12/OR_ABI-L1b-RadC-M3C01_G16_s20000011200000_e20000011200000_c20170752234173.nc
    https://noaa-goes16.s3.amazonaws.com/ABI-L1b-RadC/2000/001/12/OR_ABI-L1b-RadC-M3C01_G16_s20000011200000_e20000011200000_c20170901216521.nc
    https://noaa-goes16.s3.amazonaws.com/ABI-L1b-RadC/2000/001/12/OR_ABI-L1b-RadC-M3C01_G16_s20000011200000_e20000011200000_c20170951807462.nc
    https://noaa-goes16.s3.amazonaws.com/ABI-L1b-RadC/2000/001/12/OR_ABI-L1b-RadC-M3C01_G16_s20000011200000_e20000011200000_c20171000619157.nc
    https://noaa-goes16.s3.amazonaws.com/ABI-L1b-RadC/2000/001/12/OR_ABI-L1b-RadC-M3C01_G16_s20000011200000_e20000011200000_c20171061215161.nc


### Extracting Links from Job Output

From the output of the job we ran above, we extracted the links that we want.next is to save them to IPFS using Bacalhau.

Selecting the first ten links


```bash
%%bash
head -10 results/combined_results/stdout > links.txt
```

Selecting all the links

```
cat results/combined_results/stdout > links.txt
```

Creating a script to submit jobs


```python
%%writefile move.sh
#!/usr/bin/env bash
while read URL; do
  bacalhau docker run --input-urls="${URL}" \
  --id-only \
  --wait \
  docker.io/bacalhauproject/uploader:v0.9.14
done < links.txt
```

    Overwriting move.sh


Running the script


```bash
%%bash
bash move.sh
```

    c5c0b6dd-ce86-4b19-b666-43e3ed6fb0b4
    0a599b27-3063-46a4-82ae-244e653e0187
    2c8b7427-ee96-49b4-9516-c8596669b15f
    2cd130c1-c007-4715-a3e5-6c2d81456c09
    8c68e7be-5f85-4f2e-9cb8-3c2bb91748ae
    2850f638-6541-4ee4-9c4a-9d650699671f
    d6fb611c-a5c8-4515-9fae-53f7c7a0cfec
    6e453d0e-0baf-4905-9fa8-5ce54e5d4b65
    8177fe99-920d-4410-9cc6-bd9d0bf70f8e
    9c1acb25-6fec-4d14-a91a-4a1f60f985b9


### List the Outputs of the Jobs in JSON Format

In this case, we will move the first 10 URLs and set the no of jobs to 10 `-n 10`. If you have submitted the whole list you can set `-n` to 1000


```bash
%%bash
bacalhau list -n 10 --output json > output.json
```

Installing jq to extract CID from the results


```bash
%%bash
sudo apt update
sudo apt install jq
```

Extracting the CIDs from output json


```bash
%%bash
jq '.[] ."JobState" ."Nodes"' output.json > output-shards.json
jq '.[]."Shards"."0"."PublishedResults"."CID" | select( . != null )'  output-shards.json
```

    "QmV2uYcS7TqQGDvsLnoC2yn1inKoec9vVyTa548Gg6VTkr"
    "QmaZXQSxFDMjneyCv7ZjXdgWTNbLwPRmSEy3PMPjByeQZw"
    "QmQkafCQoSCevLN6hJKCJYRK67z3VEsFWk7qSq85GW9NUt"
    "QmZFzHeACRcqfPwTCzCfsikDLixX1NdBXCG6RHH1iiuCiY"
    "QmdZQ8vmzWRuzn9jVgzRxKnBhLsX1TQwvfT6QZdNDzcCsR"
    "QmVTL12jSTNR62zyM8zX7jVSCp1Mb5B2PUV1xkct4vo1SP"
    "QmaN5p8zteJ868cbmThTHd4yumB5eetWxXoLbcP4hWBzF1"
    "Qme3kw2tbNfmFPHXydDK9dKLzwfry8b2dxD5s4L1ij9QAL"
    "QmYki5KZQHroo1zzYWfPYrnNRDec8MVjkrvSRBCQqMzvHY"
    "QmNjarM2oxMPwN4cpQcy6NhuNbe4opHyfdce149oYkasjG"


## Need Support?

For questions, feedback, please reach out in our [forum](https://github.com/filecoin-project/bacalhau/discussions)
