---
sidebar_label: "csv-to-avro-or-parquet"
sidebar_position: 2
---
# Convert CSV To Parquet Or Avro


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

## Introduction

Converting from csv to parquet or avro reduces the size of file and allows for faster read and write speeds. With Bacalhau, you can convert your csv files stored on ipfs or on the web without the need to download files and install dependencies locally.

In this example tutorial we will convert a csv file from a url to parquet format and save the converted parquet file to IPFS

## TD;LR
Converting CSV stored in public storage with Bacalhau


## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Running CSV to Arvo or Parquet Locally​


Installing dependencies



```bash
%%bash
git clone https://github.com/js-ts/csv_to_avro_or_parquet/
```

    fatal: destination path 'csv_to_avro_or_parquet' already exists and is not an empty directory.



    ---------------------------------------------------------------------------

    CalledProcessError                        Traceback (most recent call last)

    Cell In[1], line 1
    ----> 1 get_ipython().run_cell_magic('bash', '', 'git clone https://github.com/js-ts/csv_to_avro_or_parquet/\n')


    File ~/.pyenv/versions/3.11.1/lib/python3.11/site-packages/IPython/core/interactiveshell.py:2430, in InteractiveShell.run_cell_magic(self, magic_name, line, cell)
       2428 with self.builtin_trap:
       2429     args = (magic_arg_s, cell)
    -> 2430     result = fn(*args, **kwargs)
       2432 # The code below prevents the output from being displayed
       2433 # when using magics with decodator @output_can_be_silenced
       2434 # when the last Python token in the expression is a ';'.
       2435 if getattr(fn, magic.MAGIC_OUTPUT_CAN_BE_SILENCED, False):


    File ~/.pyenv/versions/3.11.1/lib/python3.11/site-packages/IPython/core/magics/script.py:153, in ScriptMagics._make_script_magic.<locals>.named_script_magic(line, cell)
        151 else:
        152     line = script
    --> 153 return self.shebang(line, cell)


    File ~/.pyenv/versions/3.11.1/lib/python3.11/site-packages/IPython/core/magics/script.py:305, in ScriptMagics.shebang(self, line, cell)
        300 if args.raise_error and p.returncode != 0:
        301     # If we get here and p.returncode is still None, we must have
        302     # killed it but not yet seen its return code. We don't wait for it,
        303     # in case it's stuck in uninterruptible sleep. -9 = SIGKILL
        304     rc = p.returncode or -9
    --> 305     raise CalledProcessError(rc, cell)


    CalledProcessError: Command 'b'git clone https://github.com/js-ts/csv_to_avro_or_parquet/\n'' returned non-zero exit status 128.



```python
pip3 install -r csv_to_avro_or_parquet/requirements.txt
```


```python
%cd csv_to_avro_or_parquet
```

Downloading the test dataset



```python
!wget https://raw.githubusercontent.com/js-ts/csv_to_avro_or_parquet/master/movies.csv  
```

Running the conversion script arguments


```bash
%%bash
python3 src/converter.py ./movies.csv  ./movies.parquet parquet

# python converter.py path_to_csv path_to_result_file extension
```


```bash
%%bash
pip install pandas
```

Viewing the parquet file


```python
import pandas as pd
pd.read_parquet('./movies.parquet').head()
```

## Containerize Script with Docker

:::info
You can skip this section entirely and directly go to running on bacalhau
:::

To build your own docker container, create a `Dockerfile`, which contains instructions to build your image.

```
FROM python:3.8

RUN apt update && apt install git

RUN git clone https://github.com/js-ts/Sparkov_Data_Generation/

WORKDIR /Sparkov_Data_Generation/

RUN pip3 install -r requirements.txt
```

:::info
See more information on how to containerize your script/app[here](https://docs.docker.com/get-started/02_our_app/)
:::


### Build the container

We will run `docker build` command to build the container;

```
docker build -t <hub-user>/<repo-name>:<tag> .
```

Before running the command replace;

- **hub-user** with your docker hub username, If you don’t have a docker hub account [follow these instructions to create docker account](https://docs.docker.com/docker-id/), and use the username of the account you created

- **repo-name** with the name of the container, you can name it anything you want

- **tag** this is not required but you can use the latest tag

In our case:

```
docker build -t jsacex/csv-to-arrow-or-parquet
```

### Push the container

Next, upload the image to the registry. This can be done by using the Docker hub username, repo name or tag.

```
docker push <hub-user>/<repo-name>:<tag>
```

In our case:

```
docker push jsacex/csv-to-arrow-or-parquet
```

## Running a Bacalhau Job

To submit a job, we are going to either mount the script from a IPFS or from an URL.

### Mounting the CSV File from IPFS

With the command below, we are gmounting the CSV file for transactions from IPFS


```bash
%%bash --out job_id
bacalhau docker run \
-i ipfs://QmTAQMGiSv9xocaB4PUCT5nSBHrf9HZrYj21BAZ5nMTY2W  \
--wait \
--id-only \
jsacex/csv-to-arrow-or-parquet \
-- python3 src/converter.py ../inputs/transactions.csv  ../outputs/transactions.parquet parquet
```

### Structure of the command

Let's look closely at the command above:

* `bacalhau docker run`: call to bacalhau 
  
* `-i ipfs://QmTAQMGiSv9xocaB4PUCT5nSBHrf9HZrYj21BAZ5nMTY2W`: CIDs to use on the job. Mounts them at '/inputs' in the execution.

* `jsacex/csv-to-arrow-or-parque`: the name and the tag of the docker image we are using

* `../inputs/movies.csv `: path to input dataset

* `../outputs/movies.parquet parquet`: path to output

* `python3 src/converter.py`: execute the script

### Mounting the CSV File from an URL

```
bacalhau docker run \
-i https://raw.githubusercontent.com/js-ts/csv_to_avro_or_parquet/master/movies.csv   
jsacex/csv-to-arrow-or-parquet \
-- python3 src/converter.py ../inputs/movies.csv  ../outputs/movies.parquet parquet
```

### Structure of the command

Let's look closely at the command above:

* `bacalhau docker run`: call to bacalhau 
  
* `-i https://raw.githubusercontent.com/js-ts/csv_to_avro_or_parquet/master/movies.csv`: URL:path of the input data volumes downloaded from a URL source

* `jsacex/csv-to-arrow-or-parque`: the name and the tag of the docker image we are using

* `../inputs/movies.csv `: path to input dataset

* `../outputs/movies.parquet parquet`: path to output

* `python3 src/converter.py`: execute the script

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%env JOB_ID={job_id}
```

## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```


When it says `Published` or `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe ${JOB_ID}
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

## Viewing your Job Output

To view the file, run the following command:


```bash
%%bash
ls results/outputs
```

Alternatively, you can do this.


```python
import pandas as pd
import os
pd.read_parquet('results/outputs/transactions.parquet')
```

## Need Support?

For questions, feedback, please reach out in our [forum](https://github.com/filecoin-project/bacalhau/discussions)
