---
sidebar_label: "DuckDB"
sidebar_position: 1
---


DuckDB is an embedded SQL database tool that is designed to analyze data without external dependencies or state, that can be embedded locally on any machine.

Because DuckDB allows you to process and store data such as Parquet files and text logs, DuckDB can be an invaluable tool in analyzing system created data such as logs while still allowing you to use SQL as a first-class way to interact with it.

However, many organizations only want to present DuckDB on local interfaces for security and compliance  purposes, so having a central system that can interact with embedded DuckDBs would not be acceptable. Bacalhau + DuckDB provides a distributed way to execute many queries against local logs, without having to move the files at all.

## Problem

With data being generated everywhere, it can be a challenge to centralize and process the information before getting insights. Moving data to a lake can be time consuming, costly, and insecure; often just moving the data risks enormous data protection fines.

Further, the sheer number of log files alone being generated from servers, IoT devices, embedded machines, and more present a huge surface area for managing generated data. As files are written to a local data store, organizations are faced with either building remote connectivity tooling to access the files in place, or pushing these files into a data lake costing both time and money.

Ideally, a users would be able to gain insights from the remote files WITHOUT having to centralize first. This is where Bacalhau and DuckDB can step in.

## Using Bacalhau to Execute DuckDB Processing

In order to speed results and deliver more cost-effective processing of log files generated, we can use Bacalhau and DuckDB to run directly to the nodes.

The flow looks like the following:

1. Execute a command against the network to execute “local to machine” queries against the set of nodes with log files on them
2. Return the results of the queries that require immediate action (e.g. emergency alerts)
3. Archive the existing logs into cold storage.

This is laid out in the architecture below.



## Tools Used

- DuckDB
- Docker
- Python
- Terraform
- gcloud CLI

### Try out Thing

Follow the steps below to set up log processing and storage for 3 VMs in different regions or zones these VMs produce logs:

#### Step 1: Set up a “fake log creating” job


- Output something that looks like real logs -
- It should be compatible with this - [https://tersesystems.com/blog/2023/03/04/ad-hoc-structured-log-analysis-with-sqlite-and-duckdb/](https://tersesystems.com/blog/2023/03/04/ad-hoc-structured-log-analysis-with-sqlite-and-duckdb/)
- Each fake log entry should look something like this:

```JSON
    {
    "id": "<UNIQUE ID>",
    "@timestamp": "<TIME STAMP IN ISO9660>",
    "@version": "1.1",
    "message": "ServiceName [Category] Message",
    }
```

- For service name - just use one of “Auth”, “AppStack”, “Database” - each one should produce one per 5 seconds
- For category, select one from [INFO], [WARN], [CRITICAL], [SECURITY]
- For message - just output a random combination of words from a [word list](https://github.com/dwyl/english-words/files/3086945/clean_words_alpha.txt) - so each message should be like “dog cheese cow car sky”. Have it be 5 words each.
- This needs to be running reliably - so have the script run in system.d

#### Step 2 Configure logrotate on the machine


- Create a new logrotate configuration file at **`/etc/logrotate.d/my_logs`** with the content:

```bash
/path/to/logs/*.log {
hourly
rotate 1
missingok
notifempty
compress
}
```

- Each time the log rotates - put it into a special directory **`/var/logs/raw_logs`** or something. (this is a setting in log rotate - where you output the rotate to)

#### Step 3: The Bacalhau Job


- On a second machine, once per hour, trigger a job to run across all nodes identified across regions
- Pass the log path to the job spec. (Use the local mount feature (can’t use it currently))
- This job should do the following:
  - If the file is not present in raw_logs, write information to stdout: `“{ warning: raw_logs_not_found, date: <-ISO9660 Timestamp->}”` - and quit
  - If file is present:

#### **Step 3a: Use DuckDB to process the logs:**

- Use a container (like from our existing example) that has DuckDB inside it - [https://docs.bacalhau.org/examples/data-engineering/DuckDB/](https://docs.bacalhau.org/examples/data-engineering/DuckDB/)
- We should NOT use David Gasquez’s current one - we should use the generic one.
- Inside the container, use a command that loads the file - e.g. `“duckdb -s "select count(*) from '0_yellow_taxi_trips.parquet'”`
- Except, we want to select only a subset of the files e.g. `“duckdb -s "select count(*) from '0_yellow_taxi_trips.parquet' contains('abc','a')”`
- Output the match to a file on the disk - `/tmp/Region-Zone-NodeName-Security-yyyymmddhhmm.json`

#### **Step 3b - Compress the file:**

- `/tmp/Region-Zone-NodeName-SECURITY-yyyymmddhhmm.json.gz`

#### **Step 3c - Push the file to an S3 bucket:**

- Push the processed logs to s3 (s3 push functionality isn’t implemented yet - just use a standard aws CLI command - figure out with Walid how to do credentials)

#### **Step 3d - Compress the raw log file**

- `/tmp/Region-Zone-NodeName-RAW-yyyymmddhhmm.json.gz`

#### **Step 3e - Push the compressed raw log to Iceberg**

- Just use standard Iceberg API - talk with Walid about

#### **Step 3f:**

- Delete the raw log file
