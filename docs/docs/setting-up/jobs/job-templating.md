---
sidebar_label: 'Job Templates'
sidebar_position: 3
title: 'Job Templates'
description: Templating Support in Bacalhau Job Run
---


## Overview

This documentation introduces templating support for [`bacalhau job run`](../../dev/cli-reference/cli/job/run), providing users with the ability to dynamically inject variables into their job specifications. This feature is particularly useful when running multiple jobs with varying parameters such as DuckDB query, S3 buckets, prefixes, and time ranges without the need to edit each job specification file manually.

## Motivation

The motivation behind this feature arises from the need to streamline the process of preparing and running multiple jobs with different configurations. Rather than manually editing job specs for each run, users can leverage placeholders and pass actual values at runtime.

## Templating Implementation

The templating functionality in Bacalhau is built upon the Go `text/template` package. This powerful library offers a wide range of features for manipulating and formatting text based on template definitions and input variables.

For more detailed information about the Go `text/template` library and its syntax, please refer to the official documentation: [Go `text/template` Package](https://pkg.go.dev/text/template).

## Usage Examples

### Sample Job Spec (job.yaml)

```yaml
Name: docker job
Type: batch
Count: 1
Tasks:
  - Name: main
    Engine:
      Type: docker
      Params:
        Image: ubuntu:latest
        Entrypoint:
          - /bin/bash
        Parameters:
          - -c
          - echo {{.greeting}} {{.name}}
```

### Running with Templating:

```bash
bacalhau job run job.yaml --template-vars "greeting=Hello,name=World"
```

### Defining Flag Multiple Times:

```bash
bacalhau job run job.yaml --template-vars "greeting=Hello" --template-vars "name=World"
```

### Disabling Templating:

```bash
bacalhau job run job.yaml --no-template
```

### Using Environment Variables:

You can also use environment variables for templating:

```bash
export greeting=Hello
export name=World
bacalhau job run job.yaml --template-envs "*"
```

### Passing A Subset of Environment Variables:

```bash
bacalhau job run job.yaml --template-envs "greeting|name"
```

### Dry Run to Preview Templated Spec:

To preview the final templated job spec without actually submitting the job, you can use the `--dry-run` flag:

```bash
bacalhau job run job.yaml --template-vars "greeting=Hello,name=World" --dry-run
```

This will output the processed job specification, showing you how the placeholders have been replaced with the provided values.

## More Examples
### Query Live Logs
```yaml
Name: Live logs processing
Type: ops
Tasks:
  - Name: main
    Engine:
      Type: docker
      Params:
        Image: expanso/nginx-access-log-processor:1.0.0
        Parameters:
          - --query
          - {{.query}}
          - --start-time
          - {{or (index . "start-time") ""}}
          - --end-time
          - {{or (index . "end-time") ""}}
    InputSources:
      - Target: /logs
        Source:
          Type: localDirectory
          Params:
            SourcePath: /data/log-orchestration/logs
```
This is an `ops` job that runs on all nodes that match the job selection criteria. It accepts duckdb `query` variable, and two optional `start-time` and `end-time` variables to define the time range for the query.

To run this job, you can use the following command:

```bash
bacalhau job run job.yaml \
  -V "query=SELECT status FROM logs WHERE status LIKE '5__'" \
  -V "start-time=-5m"
```

### Query S3 Logs
```yaml
Name: S3 logs processing
Type: batch
Count: 1
Tasks:
  - Name: main
    Engine:
      Type: docker
      Params:
        Image: expanso/nginx-access-log-processor:1.0.0
        Parameters:
          - --query
          - {{.query}}
    InputSources:
      - Target: /logs
        Source:
          Type: s3
          Params:
            Bucket: {{.AccessLogBucket}}
            Key: {{.AccessLogPrefix}}
            Filter: {{or (index . "AccessLogPattern") ".*"}}
            Region: {{.AWSRegion}}
```
This is a `batch` job that runs on a single node. It accepts duckdb `query` variable, and four other variables to define the S3 bucket, prefix, pattern for the logs and the AWS region.

To run this job, you can use the following command:

```bash
bacalhau job run job.yaml  \
    -V "AccessLogBucket=my-bucket" \
    -V "AWSRegion=us-east-1" \
    -V "AccessLogPrefix=2023-11-19-*"  \
    -V "AccessLogPattern=^[10-12].*"
```
