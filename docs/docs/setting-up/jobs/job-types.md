---
sidebar_label: 'Job Types'
sidebar_position: 4
title: 'Job Types'
description: The different job types available in Bacalhau
---

Bacalhau has recently introduced different job types in v1.1,
providing more control and flexibility over the orchestration and scheduling of those jobs - depending on their type.

Despite the differences in job types, all jobs benefit from core functionalities provided by Bacalhau, including:```

1. **Node selection** - the appropriate nodes are selected based on several criteria, including resource availability, priority and feedback from the nodes.
2. **Job monitoring** - jobs are monitored to ensure they complete, and that they stay in a healthy state.
3. **Retries** - within limits, Bacalhau will retry certain jobs a set number of times should it fail to complete successfully when requested.


## 1 Batch Jobs

Batch jobs are executed on demand, running on a specified number of Bacalhau nodes. These jobs either run until completion or until they reach a timeout. They are designed to carry out a single, discrete task before finishing.

Ideal for intermittent yet intensive data dives, for instance performing computation over large datasets before publishing the response.  This approach eliminates the continuous processing overhead, focusing on specific, in-depth investigations and computation.

## Ops Jobs

Similar to batch jobs, ops jobs have a broader reach. They are executed on all nodes that align with the job specification, but otherwise behave like batch jobs.

Ops jobs are perfect for urgent investigations, granting direct access to logs on host machines, where previously you may have had to wait for the logs to arrive at a central locartion before being able to query them. They can also be used for delivering configuration files for other systems should you wish to deploy an update to many machines at once.

## Daemon Jobs

Daemon jobs run continuously on all nodes that meet the criteria given in the job specification. Should any new compute nodes join the cluster after the job was started, and should they meet the criteria, the job will be scheduled to run on that node too.

A good application of daemon jobs is to handle continuously generated data on every compute node.  This might be from edge devices like sensors, or cameras, or from logs where they are generated. The data can then be aggregated and compressed them before sending it onwards.  For logs, the aggregated data can be relayed at regular intervals to platforms like Kafka or Kinesis, or directly to other logging services with edge devices potentially delivering results via MQTT.

## Service Jobs

Service jobs run continuously on a specified number of nodes that meet the criteria given in the job specification. Bacalhau's orchestrator selects the optimal nodes to run the job, and continuously monitors its health, performance. If required it will reschedule on other nodes.

This job type is good for long running consumers such as streaming or queueing services, or real-time event listeners.
