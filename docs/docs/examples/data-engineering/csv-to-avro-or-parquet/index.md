---
sidebar_label: "csv-to-avro-or-parquet"
sidebar_position: 2
---
# Convert CSV To Parquet Or Avro

## Introduction

Converting from CSV to parquet or avro reduces the size of the file and allows for faster read and write speeds. With Bacalhau, you can convert your CSV files stored on ipfs or on the web without the need to download files and install dependencies locally.

In this example tutorial we will convert a CSV file from a URL to parquet format and save the converted parquet file to IPFS

## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)
```
!command -v bacalhau >/dev/null 2>&1 || (export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
path=!echo $PATH
%env PATH=./:{path[0]}
```

## Running CSV to Avro or Parquet Locally​

### Downloading the CSV file

Let's download the `transactions.csv` file:
```bash
%%bash
wget https://cloudflare-ipfs.com/ipfs/QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz/transactions.csv
```

:::tip
You can use the CSV files from [here](https://github.com/datablist/sample-csv-files?tab=readme-ov-file)
:::

### Writing the Script

Write the `converter.py` Python script, that serves as a CSV converter to Avro or Parquet formats:

```python
%%writefile converter.py
import os
import sys
from abc import ABCMeta, abstractmethod

import fastavro
import numpy as np
import pandas as pd
from pyarrow import Table, parquet


class BaseConverter(metaclass=ABCMeta):
    """
    Base class for converters.

    Validate received parameters for future use.
    """
    def __init__(
        self,
        csv_file_path: str,
        target_file_path: str,
    ) -> None:
        self.csv_file_path = csv_file_path
        self.target_file_path = target_file_path

    @property
    def csv_file_path(self):
        return self._csv_file_path

    @csv_file_path.setter
    def csv_file_path(self, path):
        if not os.path.isabs(path):
            path = os.path.join(os.getcwd(), path)
        _, extension = os.path.splitext(path)
        if not os.path.isfile(path) or extension != '.csv':
            raise FileNotFoundError(
                f'No such csv file: {path}'
            )
        self._csv_file_path = path

    @property
    def target_file_path(self):
        return self._target_file_path

    @target_file_path.setter
    def target_file_path(self, path):
        if not os.path.isabs(path):
            path = os.path.join(os.getcwd(), path)
        target_dir = os.path.dirname(path)
        if not os.path.isdir(target_dir):
            raise FileNotFoundError(
                f'No such directory: {target_dir}\n'
                'Choose existing or create directory for result file.'
            )
        if os.path.isfile(path):
            raise FileExistsError(
                f'File {path} has already exists.'
                'Usage of existing file may result in data loss.'
            )
        self._target_file_path = path

    def get_csv_reader(self):
        """Return csv reader which read csv file as a stream"""
        return pd.read_csv(
            self.csv_file_path,
            iterator=True,
            chunksize=100000
        )

    @abstractmethod
    def convert(self):
        """Should be implemented in child class"""
        pass


class ParquetConverter(BaseConverter):
    """
    Convert received csv file to parquet file.

    Take path to csv file and path to result file.
    """
    def convert(self):
        """Read csv file as a stream and write data to parquet file."""
        csv_reader = self.get_csv_reader()
        writer = None
        for chunk in csv_reader:
            if not writer:
                table = Table.from_pandas(chunk)
                writer = parquet.ParquetWriter(
                    self.target_file_path, table.schema
                )
            table = Table.from_pandas(chunk)
            writer.write_table(table)
        writer.close()


class AvroConverter(BaseConverter):
    """
    Convert received csv file to avro file.

    Take path to csv file and path to result file.
    """
    NUMPY_TO_AVRO_TYPES = {
        np.dtype('?'): 'boolean',
        np.dtype('int8'): 'int',
        np.dtype('int16'): 'int',
        np.dtype('int32'): 'int',
        np.dtype('uint8'): 'int',
        np.dtype('uint16'): 'int',
        np.dtype('uint32'): 'int',
        np.dtype('int64'): 'long',
        np.dtype('uint64'): 'long',
        np.dtype('O'): ['null', 'string', 'float'],
        np.dtype('unicode_'): 'string',
        np.dtype('float32'): 'float',
        np.dtype('float64'): 'double',
        np.dtype('datetime64'): {
            'type': 'long',
            'logicalType': 'timestamp-micros'
        },
    }

    def get_avro_schema(self, pandas_df):
        """Generate avro schema."""
        column_dtypes = pandas_df.dtypes
        schema_name = os.path.basename(self.target_file_path)
        schema = {
            'type': 'record',
            'name': schema_name,
            'fields': [
                {
                    'name': name,
                    'type': AvroConverter.NUMPY_TO_AVRO_TYPES[dtype]
                } for (name, dtype) in column_dtypes.items()
            ]
        }
        return fastavro.parse_schema(schema)

    def convert(self):
        """Read csv file as a stream and write data to avro file."""
        csv_reader = self.get_csv_reader()
        schema = None
        with open(self.target_file_path, 'a+b') as f:
            for chunk in csv_reader:
                if not schema:
                    schema = self.get_avro_schema(chunk)
                fastavro.writer(
                    f,
                    schema=schema,
                    records=chunk.to_dict('records')
                )


if __name__ == '__main__':
    converters = {
        'parquet': ParquetConverter,
        'avro': AvroConverter
    }
    csv_file, result_path, result_type = sys.argv[1], sys.argv[2], sys.argv[3]
    if result_type.lower() not in converters:
        raise ValueError(
            'Invalid target type. Avalible types: avro, parquet.'
        )
    converter = converters[result_type.lower()](csv_file, result_path)
    converter.convert()
```

:::info
You can find out more information about `converter.py` [here](https://github.com/bacalhau-project/examples/blob/ef3a657336934261cdfc50b10b8981b691cbf203/data-engineering/csv-to-avro-or-parquet/csv-to-avro-parquet/README.md?plain=1#L4)
:::


### Installing Dependencies

```bash
%%bash
pip install fastavro numpy pandas pyarrow
```

### Converting CSV file to Parquet format

```shell
python converter.py <path_to_csv> <path_to_result_file> <extension>
```

In our case:

```bash
%%bash
python3 converter.py transactions.csv transactions.parquet parquet
```

### Viewing the parquet file: 

```python
import pandas as pd
pd.read_parquet('transactions.parquet').head()
```

## Containerize Script with Docker

:::info
You can skip this section entirely and directly go to running on Bacalhau
:::

To build your own docker container, create a `Dockerfile`, which contains instructions to build your image.

```
FROM python:3.8

RUN apt update && apt install git

RUN git clone https://github.com/bacalhau-project/Sparkov_Data_Generation

WORKDIR /Sparkov_Data_Generation/

RUN pip3 install -r requirements.txt
```

:::info
See more information on how to containerize your script/app [here](https://docs.docker.com/get-started/02_our_app/)
:::


### Build the container

We will run the `docker build` command to build the container:

```
docker build -t <hub-user>/<repo-name>:<tag> .
```

Before running the command replace:

**`hub-user`** with your docker hub username. If you don’t have a docker hub account [follow these instructions to create docker account](https://docs.docker.com/docker-id/), and use the username of the account you created

**`repo-name`** with the name of the container, you can name it anything you want

**`tag`** this is not required but you can use the latest tag

In our case:

```
docker build -t jsacex/csv-to-arrow-or-parquet .
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

With the command below, we are mounting the CSV file for transactions from IPFS


```python
!command -v bacalhau >/dev/null 2>&1 || (export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
path=!echo $PATH
%env PATH=./:{path[0]}
```


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

1. `bacalhau docker run`: call to Bacalhau
1. `-i ipfs://QmTAQMGiSv9xocaB4PUCT5nSBHrf9HZrYj21BAZ5nMTY2W`: CIDs to use on the job. Mounts them at '/inputs' in the execution.
1. `jsacex/csv-to-arrow-or-parque`: the name and the tag of the docker image we are using
1. `../inputs/transactions.csv `: path to input dataset
1. `../outputs/transactions.parquet parquet`: path to the output
1. `python3 src/converter.py`: execute the script

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

```python
%env JOB_ID={job_id}
```
### Declarative job description

The same job can be presented in the [declarative](../../../setting-up/jobs/job-specification/job.md) format. In this case, the description will look like this:

```yaml
name: Convert CSV To Parquet Or Avro
type: batch
count: 1
tasks:
  - name: My main task
    Engine:
      type: docker
      params:
        Image: jsacex/csv-to-arrow-or-parquet
        Entrypoint:
          - /bin/bash
        Parameters:
          - -c
          - python3 src/converter.py ../inputs/transactions.csv  ../outputs/transactions.parquet parquet
    Publisher:
      Type: ipfs
    ResultPaths:
      - Name: outputs
        Path: /outputs
    InputSources:
      - Target: "/inputs"
        Source:
          Type: "ipfs"
          Params:
            CID: "QmTAQMGiSv9xocaB4PUCT5nSBHrf9HZrYj21BAZ5nMTY2W"
```

The job description should be saved in `.yaml` format, e.g. `convertcsv.yaml`, and then run with the command:
```bash
bacalhau job run convertcsv.yaml
```

## Checking the State of your Jobs

**Job status**: You can check the status of the job using `bacalhau list`.

```bash
%%bash
bacalhau list --id-filter ${JOB_ID} 
```
Expected Output:

```shell
 CREATED   ID          JOB                                       STATE      PUBLISHED
 13:27:10  cce0f374    Type:"docker",Params:"map[Entrypoint:<ni  Completed
                       l> EnvironmentVariables:[] Image:jsacex/
                       csv-to-arrow-or-parquet Parameters:[pyth
                       on3 src/converter.py ../inputs/transacti
                       ons.csv ../outputs/transactions.parquet
                       parquet] WorkingDirectory:]"
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
rm -rf results && mkdir -p results # Temporary directory to store the results
bacalhau get ${JOB_ID} --output-dir results # Download the results
```

## Viewing your Job Output

To view the file, run the following command:

```bash
%%bash
ls results/outputs

Expected Output:
transactions.parquet
```

Alternatively, you can do this.

```python
import pandas as pd
import os
pd.read_parquet('results/outputs/transactions.parquet')
```

## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).
