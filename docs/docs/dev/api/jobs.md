---
sidebar_label: Jobs
---

# Jobs API Documentation

Job APIs enables creating, managing, monitoring, and analyzing jobs in Bacalhau.

## Describe Job

**Endpoint:** `GET /api/v1/orchestrator/jobs/:jobID`

Retrieve the specification and current status of a particular job.

**Parameters**:
  - `:jobID`: Identifier of the job to describe. This can be full ID of the job (e.g. `j-28c08f7f-6fb0-48ed-912d-a2cb6c3a4f3a`) or just the short format (e.g. `j-28c08f7f`) if it's unique.

**Response**:
- **Job**: Specification for the requested [job](../../setting-up/jobs/job-specification/job.md).

**Example**:
```bash
curl 127.0.0.1:1234/api/v1/orchestrator/jobs/j-d586d2cc-6fc9-42c4-9dd9-a78df1d7cd01
{
  "Job": {
    "ID": "j-d586d2cc-6fc9-42c4-9dd9-a78df1d7cd01",
    "Name": "A sample job",
    "Namespace": "default",
    "Type": "batch",
    "Priority": 0,
    "Count": 1,
    "Constraints": [],
    "Meta": {
      "bacalhau.org/requester.id": "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
      "bacalhau.org/requester.publicKey": "CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDVRKPgCfY2fgfrkHkFjeWcqno+MDpmp8DgVaY672BqJl/dZFNU9lBg2P8Znh8OTtHPPBUBk566vU3KchjW7m3uK4OudXrYEfSfEPnCGmL6GuLiZjLf+eXGEez7qPaoYqo06gD8ROdD8VVse27E96LlrpD1xKshHhqQTxKoq1y6Rx4DpbkSt966BumovWJ70w+Nt9ZkPPydRCxVnyWS1khECFQxp5Ep3NbbKtxHNX5HeULzXN5q0EQO39UN6iBhiI34eZkH7PoAm3Vk5xns//FjTAvQw6wZUu8LwvZTaihs+upx2zZysq6CEBKoeNZqed9+Tf+qHow0P5pxmiu+or+DAgMBAAE="
    },
    "Labels": {
      "env": "prod",
      "name": "demo"
    },
    "Tasks": [
      {
        "Name": "main",
        "Engine": {
          "Type": "docker",
          "Params": {
            "Entrypoint": [
              "/bin/bash"
            ],
            "Image": "ubuntu:latest",
            "Parameters": [
              "-c",
              "echo hello world"
            ]
          }
        },
        "Publisher": {
          "Type": "",
          "Params": {}
        },
        "Env": {},
        "Meta": {},
        "InputSources": [],
        "ResultPaths": [],
        "Resources": {
          "CPU": "",
          "Memory": "",
          "Disk": "",
          "GPU": ""
        },
        "Network": {
          "Type": "None"
        },
        "Timeouts": {
          "ExecutionTimeout": 1800
        }
      }
    ],
    "State": {
      "StateType": "Completed",
      "Message": ""
    },
    "Version": 0,
    "Revision": 2,
    "CreateTime": 1695883778909107178,
    "ModifyTime": 1695883779369191994
  }
}
```

## List Jobs

**Endpoint:** `GET /api/v1/orchestrator/jobs`

Retrieve a list of jobs.

**Parameters**:
  - `namespace`: Specify a namespace to filter the jobs. Use `*` to display jobs from all namespaces.
  - `labels`: Use label-based criteria to filter jobs. See [Label Filtering](../api) for usage details.
  - `limit`: Set the maximum number of jobs to return. Default is set to 10.
  - `next_token`: Utilize this parameter for pagination continuation.
  - `order_by`: Determine the ordering of jobs. Choose between `id` or `create_time` (default is `create_time`).
  - `reverse`: Opt to reverse the default order of displayed jobs.

**Response**:
- **[Jobs](../../setting-up/jobs/job-specification/job.md)**: List of matching jobs.
- **NextToken** `(string)`: Pagination token.

**Example**:

List jobs with limit set to 3:
```bash
curl 127.0.0.1:1234/api/v1/orchestrator/jobs?limit=3
{
  "Jobs": [
    {
      "ID": "j-f6331e9a-727d-4175-8350-095b6b372408",
      # ...
    },
    {
      "ID": "j-f7853204-a553-4991-a1a3-816b88fdbfc7",
      # ...
    },
    {
      "ID": "j-f791ad14-af5b-4c26-8c93-15cc23dca811",
      # ...
    }
  ],
  "NextToken": ""
}
```

List with label filtering
```bash
curl --get 127.0.0.1:1234/api/v1/orchestrator/jobs --data-urlencode 'labels=env in (prod,dev)'
```

## Create Job

**Endpoint:** `PUT /api/v1/orchestrator/jobs`

Submit a new job for execution.

**Request Body**:
  - **[Job](../../setting-up/jobs/job-specification/job.md)**: JSON definition of the job.

**Response**:
- **JobID** `(string)`: Identifier for the new job.
- **EvaluationID** `(string)`: Identifier for the evaluation to schedule the job.
- **Warnings** `(string[])`: Any warnings during job submission.

**Example**:
```bash
curl -X PUT \
     -H "Content-Type: application/json" \
     -d '{
          "Job": {
            "Name": "test-job",
            "Type": "batch",
            "Count": 1,
            "Labels": {
              "foo": "bar",
              "env": "dev"
            },
            "Tasks": [
              {
                "Name": "task1",
                "Engine": {
                  "Type": "docker",
                  "Params": {
                    "Image": "ubuntu:latest",
                    "Entrypoint": [
                      "echo",
                      "hello"
                    ]
                  }
                },
                "Publisher": {
                  "Type": "noop"
                }
              }
            ],
            "CreateTime": 1234
          }
        }' \
     127.0.0.1:1234/api/v1/orchestrator/jobs

 {
  "JobID": "j-9809ae4b-d4fa-47c6-823b-86c924e60604",
  "EvaluationID": "5dac9fe0-2358-4ec7-bec9-6747dfa2b33e",
  "Warnings": [
    "job create time is ignored when submitting a job"
  ]
}
```

## Stop Job

**Endpoint:** `DELETE /api/v1/orchestrator/jobs/:jobID`

Terminate a specific job asynchronously.

**Parameters**:
  - `:jobID`: Identifier of the job to describe. This can be full ID of the job (e.g. `j-28c08f7f-6fb0-48ed-912d-a2cb6c3a4f3a`) or just the short format (e.g. `j-28c08f7f`) if it's unique.
  - `reason`: A message for debugging and traceability.

**Response**:
- **EvaluationID** `(string)`: Identifier for the evaluation to stop the job.

**Example**:
```bash
curl -X DELETE 127.0.0.1:1234/api/v1/orchestrator/jobs/j-50ee38d5-2812-4365-aceb-7b47b8f3858e
{
  "EvaluationID": "1316fdfe-97c4-43bc-8e0b-50a7f02f18bb"
}
```

## Job History

**Endpoint:** `GET /api/v1/orchestrator/jobs/:jobID/history`

Retrieve historical events for a specific job.

**Parameters**:
  - `since`: Timestamp to start (default: 0).
  - `event_type`: Filter by event type: `job`, `execution`, or `all` (default).
  - `execution_id`: Filter by execution ID.
  - `node_id`: Filter by node ID.
  - `limit`: Maximum events to return.
  - `next_token`: For pagination.

**Response**:
- **History**: List of matching historical events.
- **NextToken** `(string)`: Pagination token.

**Example**:

List events for a specific execution
```bash
curl 127.0.0.1:1234/api/v1/orchestrator/jobs/j-4cd1566f-84cb-4830-a96b-1349f5b54b1b/history\?execution_id=e-82f7813f-58da-4323-8261-886af35284c4
{
  "NextToken": "",
  "History": [
    {
      "Type": "ExecutionLevel",
      "JobID": "j-4cd1566f-84cb-4830-a96b-1349f5b54b1b",
      "NodeID": "QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT",
      "ExecutionID": "e-82f7813f-58da-4323-8261-886af35284c4",
      "JobState": null,
      "ExecutionState": {
        "Previous": 1,
        "New": 1
      },
      "NewRevision": 1,
      "Comment": "",
      "Time": "2023-09-28T07:23:01.352803607Z"
    },
    {
      "Type": "ExecutionLevel",
      "JobID": "j-4cd1566f-84cb-4830-a96b-1349f5b54b1b",
      "NodeID": "QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT",
      "ExecutionID": "e-82f7813f-58da-4323-8261-886af35284c4",
      "JobState": null,
      "ExecutionState": {
        "Previous": 1,
        "New": 2
      },
      "NewRevision": 2,
      "Comment": "",
      "Time": "2023-09-28T07:23:01.446196661Z"
    },
    {
      "Type": "ExecutionLevel",
      "JobID": "j-4cd1566f-84cb-4830-a96b-1349f5b54b1b",
      "NodeID": "QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT",
      "ExecutionID": "e-82f7813f-58da-4323-8261-886af35284c4",
      "JobState": null,
      "ExecutionState": {
        "Previous": 2,
        "New": 3
      },
      "NewRevision": 3,
      "Comment": "",
      "Time": "2023-09-28T07:23:01.604862596Z"
    },
    {
      "Type": "ExecutionLevel",
      "JobID": "j-4cd1566f-84cb-4830-a96b-1349f5b54b1b",
      "NodeID": "QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT",
      "ExecutionID": "e-82f7813f-58da-4323-8261-886af35284c4",
      "JobState": null,
      "ExecutionState": {
        "Previous": 3,
        "New": 3
      },
      "NewRevision": 4,
      "Comment": "",
      "Time": "2023-09-28T07:23:01.611816334Z"
    },
    {
      "Type": "ExecutionLevel",
      "JobID": "j-4cd1566f-84cb-4830-a96b-1349f5b54b1b",
      "NodeID": "QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT",
      "ExecutionID": "e-82f7813f-58da-4323-8261-886af35284c4",
      "JobState": null,
      "ExecutionState": {
        "Previous": 3,
        "New": 5
      },
      "NewRevision": 5,
      "Comment": "",
      "Time": "2023-09-28T07:23:01.705013737Z"
    },
    {
      "Type": "ExecutionLevel",
      "JobID": "j-4cd1566f-84cb-4830-a96b-1349f5b54b1b",
      "NodeID": "QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT",
      "ExecutionID": "e-82f7813f-58da-4323-8261-886af35284c4",
      "JobState": null,
      "ExecutionState": {
        "Previous": 5,
        "New": 7
      },
      "NewRevision": 6,
      "Comment": "",
      "Time": "2023-09-28T07:23:02.483265228Z"
    }
  ]
}
```
## Job Executions

**Endpoint:** `GET /api/v1/orchestrator/jobs/:jobID/executions`

Retrieve all executions for a particular job.

**Parameters**:
  - `limit`: Maximum executions to return.
  - `next_token`: For pagination.
  - `order_by`: Order by `modify_time` (default), `create_time`, `id`, `state`.
  - `reverse`: Reverse the order.

**Response**:
- **Executions**: List of relevant executions.
- **NextToken** `(string)`: Pagination token.

**Example**

List executions for a batch job with 3 executions (i.e. `count=3`)
```bash
curl 127.0.0.1:1234/api/v1/orchestrator/jobs/j-412c34b4-da77-4a46-886c-76e03615a04e/executions
{
  "NextToken": "",
  "Executions": [
    {
      "ID": "e-cdd9fb3e-3183-4069-8bc9-679b6bcce4db",
      "Namespace": "default",
      "EvalID": "",
      "Name": "",
      "NodeID": "QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
      "JobID": "j-412c34b4-da77-4a46-886c-76e03615a04e",
      "AllocatedResources": {
        "Tasks": {}
      },
      "DesiredState": {
        "StateType": 2,
        "Message": "execution completed"
      },
      "ComputeState": {
        "StateType": 7,
        "Message": ""
      },
      "PublishedResult": {
        "Type": "",
        "Params": null
      },
      "RunOutput": {
        "stdout": "hello world\n",
        "stdouttruncated": false,
        "stderr": "",
        "stderrtruncated": false,
        "exitCode": 0,
        "runnerError": ""
      },
      "PreviousExecution": "",
      "NextExecution": "",
      "FollowupEvalID": "",
      "Revision": 6,
      "CreateTime": 1695886565851709698,
      "ModifyTime": 1695886566370340241
    },
    {
      "ID": "e-836a4a50-f6cd-479f-a20d-2a12ff7fea64",
      "Namespace": "default",
      "EvalID": "",
      "Name": "",
      "NodeID": "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
      "JobID": "j-412c34b4-da77-4a46-886c-76e03615a04e",
      "AllocatedResources": {
        "Tasks": {}
      },
      "DesiredState": {
        "StateType": 2,
        "Message": "execution completed"
      },
      "ComputeState": {
        "StateType": 7,
        "Message": ""
      },
      "PublishedResult": {
        "Type": "",
        "Params": null
      },
      "RunOutput": {
        "stdout": "hello world\n",
        "stdouttruncated": false,
        "stderr": "",
        "stderrtruncated": false,
        "exitCode": 0,
        "runnerError": ""
      },
      "PreviousExecution": "",
      "NextExecution": "",
      "FollowupEvalID": "",
      "Revision": 6,
      "CreateTime": 1695886565855906980,
      "ModifyTime": 1695886566505560693
    },
    {
      "ID": "e-b7e7adc7-b28c-4af0-9002-a7fdce303634",
      "Namespace": "default",
      "EvalID": "",
      "Name": "",
      "NodeID": "QmUDAXvv31WPZ8U9CzuRTMn9iFGiopGE7rHiah1X8a6PkT",
      "JobID": "j-412c34b4-da77-4a46-886c-76e03615a04e",
      "AllocatedResources": {
        "Tasks": {}
      },
      "DesiredState": {
        "StateType": 2,
        "Message": "execution completed"
      },
      "ComputeState": {
        "StateType": 7,
        "Message": ""
      },
      "PublishedResult": {
        "Type": "",
        "Params": null
      },
      "RunOutput": {
        "stdout": "hello world\n",
        "stdouttruncated": false,
        "stderr": "",
        "stderrtruncated": false,
        "exitCode": 0,
        "runnerError": ""
      },
      "PreviousExecution": "",
      "NextExecution": "",
      "FollowupEvalID": "",
      "Revision": 6,
      "CreateTime": 1695886565853878926,
      "ModifyTime": 1695886566583711985
    }
  ]
}
```

## Job Results

**Endpoint:** `GET /api/v1/orchestrator/jobs/:jobID/results`

Fetch results published by all executions for the defined job. Applicable only for `batch` and `ops` jobs.

**Response**:
- **[Results](../../setting-up/jobs/job-specification/spec-config.md)**: List of all published results.
- **NextToken** `(string)`: Pagination token.

**Example**:

Result of a job that used the [S3 Publisher](../../setting-up/other-specifications/publishers/s3):
```bash
curl 127.0.0.1:1234/api/v1/orchestrator/jobs/j-479d160f-f9ab-4e32-aec9-a45554126450/results
{
  "NextToken": "",
  "Results": [
    {
      "Type": "s3",
      "Params": {
        "Bucket": "bacalhau-test-datasets",
        "Key": "my-prefix/my-result-file.tar.gz",
        "Region": "eu-west-1",
        "ChecksumSHA256": "qKAFvkLvSc+QqHE4hFiy4qVEmXhr423lQaRBfJecsgo=",
        "VersionID": "bNS92VdFudVI7NPsXF51Qn.RPw31TKNG"
      }
    }
  ]
}
```
