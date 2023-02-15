---
sidebar_label: "csv-to-avro-or-parquet"
sidebar_position: 10
---
# Convert CSV To Parquet Or Arrow

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-engineering/csv-to-avro-or-parquet/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-engineering/csv-to-avro-or-parquet/index.ipynb)

## Introduction

Converting from csv to parquet or avro reduces the size of file and allows for faster read and write speeds, using bacalhau you can convert your csv files stored on ipfs or on the web without
The need to download files and install dependencies locally

In this example we will convert a csv file from a url to parquet format and save the converted parquet file to IPFS


## Running Locally​


Installing dependencies



```bash
%%bash
git clone https://github.com/js-ts/csv_to_avro_or_parquet/
pip3 install -r csv_to_avro_or_parquet/requirements.txt
```


```python
%cd csv_to_avro_or_parquet
```

Downloading the test dataset



```python
!wget https://raw.githubusercontent.com/js-ts/csv_to_avro_or_parquet/master/movies.csv  
```

Running the conversion script

arguments
```
python converter.py path_to_csv path_to_result_file extension
```

Running the script





```bash
%%bash
python3 src/converter.py ./movies.csv  ./movies.parquet parquet
```

viewing the parquet file


```python
import pandas as pd
pd.read_parquet('./movies.parquet').head()
```


title	rating	year	runtime
0	Almost Famous	R	2000	122
1	American Pie	R	1999	95
2	Back to the Future	PG	1985	116
3	Blade Runner	R	1982	117
4	Blood for Dracula	R	1974	106

### Building a Docker container (Optional)
Note* you can skip this section entirely and directly go to running on bacalhau

To use Bacalhau, you need to package your code in an appropriate format. The developers have already pushed a container for you to use, but if you want to build your own, you can follow the steps below. You can view a [dedicated container example](https://docs.bacalhau.org/examples/workload-onboarding/custom-containers/) in the documentation.

### Dockerfile

In this step, you will create a `Dockerfile` to create an image. The `Dockerfile` is a text document that contains the commands used to assemble the image. First, create the `Dockerfile`.

```
FROM python:3.8

RUN apt update && apt install git

RUN git clone https://github.com/js-ts/Sparkov_Data_Generation/

WORKDIR /Sparkov_Data_Generation/

RUN pip3 install -r requirements.txt
```

To Build the docker container run the docker build command

```
docker build -t hub-user/repo-name:tag .
```

Please replace

hub-user with your docker hub username, If you don’t have a docker hub account Follow these instructions to create docker account, and use the username of the account you created

repo-name This is the name of the container, you can name it anything you want

tag This is not required but you can use the latest tag

After you have build the container, the next step is to test it locally and then push it docker hub

Now you can push this repository to the registry designated by its name or tag.

```
 docker push hub-user/repo-name:tag
```


After the repo image has been pushed to docker hub, we can now use the container for running on bacalhau

## Running on Bacalhau

After the repo image has been pushed to docker hub, we can now use the container for running on bacalhau

This command is similar to what we have run locally but we change the output directory to the outputs folder so that the results are saved to IPFS

we will show you how you can mount the script from a IPFS as we as from an URL


```bash
%%bash
curl -sL https://get.bacalhau.org/install.sh | bash
```

    Your system is linux_amd64
    No BACALHAU detected. Installing fresh BACALHAU CLI...
    Getting the latest BACALHAU CLI...
    Installing v0.3.11 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.3.11/bacalhau_v0.3.11_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.3.11/bacalhau_v0.3.11_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into /usr/local/bin successfully.
    Client Version: v0.3.11
    Server Version: v0.3.11


Mounting the csv file from IPFS


```bash
%%bash --out job_id
bacalhau docker run \
-i QmTAQMGiSv9xocaB4PUCT5nSBHrf9HZrYj21BAZ5nMTY2W  \
--wait \
--id-only \
 jsacex/csv-to-arrow-or-parquet \
-- python3 src/converter.py ../inputs/transactions.csv  ../outputs/transactions.parquet parquet
```

Mounting the csv file from an URL

```
bacalhau docker run \
-u https://raw.githubusercontent.com/js-ts/csv_to_avro_or_parquet/master/movies.csv   jsacex/csv-to-arrow-or-parquet \
-- python3 src/converter.py ../inputs/movies.csv  ../outputs/movies.parquet parquet
```


```python
%env JOB_ID={job_id}
```

    env: JOB_ID=94774248-1d07-4121-aac8-451aca4a636e


Running the commands will output a UUID that represents the job that was created. You can check the status of the job with the following command:


```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 10:19:19 [0m[97;40m 94774248 [0m[97;40m Docker jsacex/csv-to... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmdHJaMmQHs9fE... [0m



Where it says "`Completed `", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```bash
%%bash
bacalhau describe ${JOB_ID}
```

If you see that the job has completed and there are no errors, then you can download the results with the following command:


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

    Fetching results of job '94774248-1d07-4121-aac8-451aca4a636e'...
    Results for job '94774248-1d07-4121-aac8-451aca4a636e' have been written to...
    results


    2022/11/12 10:20:09 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.


After the download has finished you should 
see the following contents in results directory


```bash
%%bash
ls results/combined_results/outputs
```

    transactions.parquet


Viewing the output


```python
import pandas as pd
import os
pd.read_parquet('results/combined_results/outputs/transactions.parquet')
```
