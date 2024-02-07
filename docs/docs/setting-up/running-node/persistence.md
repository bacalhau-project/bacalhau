---
sidebar_label: 'Configuring node persistence'
sidebar_position: 180
title: 'Configuring node persistence'
description: How to configure compute/requester persistence
---

Both compute nodes, and requester nodes, maintain state. How that state is maintained is configurable, although the defaults are likely adequate for most use-cases.  This page describes how to configure the persistence of compute and requester nodes should the defaults not be suitable.

## Compute node persistence

The computes nodes maintain information about the work that has been allocated to them, including:

* The current state of the execution, and
* The original job that resulted in this allocation

This information is used by the compute and requester nodes to ensure allocated jobs are completed successfully.  By default, compute nodes store their state in a bolt-db database and this is located in the bacalhau repository along with configuration data.  For a compute node whose ID is "abc", the database can be found in `~/.bacalhau/abc-compute/executions.db`.

In some cases, it may be preferable to maintain the state in memory, with the caveat that should the node restart, all state will be lost.  This can be configured using the environment variables in the table below.

|Environment Variable|Flag alternative|Value|Effect|
|--|--|--|--|
|BACALHAU_COMPUTE_STORE_TYPE|--compute-execution-store-type|boltdb|Uses the bolt db execution store (default)|
|BACALHAU_COMPUTE_STORE_PATH|--compute-execution-store-path|A path (inc. filename)|Specifies where the boltdb database should be stored. Default is `~/.bacalhau/{NODE-ID}-compute/executions.db` if not set|

## Requester node persistence

When running a requester node, it maintains state about the jobs it has been requested to orchestrate and schedule, the evaluation of those jobs, and the executions that have been allocated.  By default, this state is stored in a bolt db database that, with a node ID of "xyz" can be found in  `~/.bacalhau/xyz-requester/jobs.db`.


|Environment Variable|Flag alternative|Value|Effect|
|--|--|--|--|
|BACALHAU_JOB_STORE_TYPE|--requester-job-store-type|boltdb|Uses the bolt db job store (default)|
|BACALHAU_JOB_STORE_PATH|--requester-job-store-path|A path (inc. filename)|Specifies where the boltdb database should be stored. Default is `~/.bacalhau/{NODE-ID}-requester/jobs.db` if not set|
