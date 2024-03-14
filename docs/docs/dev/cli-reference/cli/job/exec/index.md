---
sidebar_label: exec
---
# Command: `exec`

## Description

The `bacalhau exec` command allows for the specification of jobs to be executed from the command line,
without the need for a job specification file (see [job run](/dev/cli-reference/cli/job/run/)).

## Usage

```shell
bacalhau exec [flags] [job-type] arguments
```

## Flags

- `-h`, `--help`:
    - Description: Displays help information for the `exec` sub-command.

- `--code`:
    - Includes the specified code in the job. This can be a single file, or a directory containing many files.  There is a limit of 10Mb on the size of the uploaded code.

- `-f`, `--follow`:
    - Description: If provided, the command will continuously display the output from the job as it runs.

- `--wait`
	- Description: Wait for the job to finish. Use --wait=false to return as soon as the job is submitted.

- `--wait-timeout-secs`
	- Description: When using --wait, how many seconds to wait for the job to complete before giving up.

- `--node-details`
	- Description: Print out details of all nodes (overridden by --id-only).

- `--id-only`:
    - Description: On successful job submission, only the Job ID will be printed.

- `-p`, `--publisher`
	- Description: Where to publish the result of the job.
	   ### Examples:
	   **Publish to IPFS**

         `-p ipfs`

       **Publish to S3**

        `-p s3://bucket/key`

- `-i`, `--input`
    - Description: Mount URIs as inputs to the job. Can be specified multiple times. Format: src=URI,dst=PATH[,opt=key=value]
        ### Examples:
        **Mount IPFS CID to /inputs directory**

        `-i ipfs://QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72`

        **Mount S3 object to a specific path**

        `-i s3://bucket/key,dst=/my/input/path`

        **Mount S3 object with specific endpoint and region**

        `-i src=s3://bucket/key,dst=/my/input/path,opt=endpoint=https://s3.example.com,opt=region=us-east-1`

- `-o`, `--output`
    - Description: name:path of the output data volumes. 'outputs:/outputs' is always added unless '/outputs' is mapped to a different name.

- `-e`, `--env`
    - Description: The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)

- `--timeout`
    - Description:  Job execution timeout in seconds (e.g. 300 for 5 minutes)

- `-l`, `--labels`
    - Description: List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.

- `-s`, `--selector`
    - Description: Selector (label query) to filter nodes on which this job can be executed, supports '=', '==', and '!='.(e.g. -s key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.


## Global Flags

- `--api-host string`:
    - Description: Specifies the host used for RESTful communication between the client and server. The flag is disregarded if the `BACALHAU_API_HOST` environment variable is set.
    - Default: `bootstrap.production.bacalhau.org`

- `--api-port int`:
    - Description: Specifies the port for REST communication. If the `BACALHAU_API_PORT` environment variable is set, this flag will be ignored.
    - Default: `1234`

- `--log-mode logging-mode`:
    - Description: Sets the desired log format. Options are: `default`, `station`, `json`, `combined`, and `event`.
    - Default: `default`

- `--repo string`:
    - Description: Defines the path to the bacalhau repository.
    - Default: ``$HOME/.bacalhau`


## Examples

### Running python tasks

1. **Basic Usage**:

   **Command**:
   ```shell
   → bacalhau exec python -- -c "import this"
   ```

   **Output**:
   ```text
   The Zen of Python, by Tim Peters

   Beautiful is better than ugly.
   Explicit is better than implicit.
   Simple is better than complex.
   Complex is better than complicated.
   Flat is better than nested.
   ....
   ```

2. **Single file Python**:

   **Command**:
   ```shell
   → bacalhau exec --code=app.py python app.py
   ```

   where app.py is
   ```python
   """
   pip install colorama
   """

   from colorama import Fore
   print(Fore.RED + "Hello World")
   ```

   **Output**:

   As red text

   ```shell
   Hello World
   ```


### Running duckdb queries

1. **Basic Usage**:

   **Command**:
   ```shell
   → cat describe.sql
     DESCRIBE TABLE '/inputs/world-cities_csv.csv';

   → bacalhau exec --code=describe.sql -i src=https://datahub.io/core/world-cities/r/world-cities.csv,dst=/inputs duckdb -- -init /code/describe.sql
   ```

   **Output**:
   ```text
        ┌─────────────┬─────────────┬─────────┬─────────┬─────────┬─────────┐
        │ column_name │ column_type │  null   │   key   │ default │  extra  │
        │   varchar   │   varchar   │ varchar │ varchar │ varchar │ varchar │
        ├─────────────┼─────────────┼─────────┼─────────┼─────────┼─────────┤
        │ name        │ VARCHAR     │ YES     │         │         │         │
        │ country     │ VARCHAR     │ YES     │         │         │         │
        │ subcountry  │ VARCHAR     │ YES     │         │         │         │
        │ geonameid   │ BIGINT      │ YES     │         │         │         │
        └─────────────┴─────────────┴─────────┴─────────┴─────────┴─────────┘
   ```
