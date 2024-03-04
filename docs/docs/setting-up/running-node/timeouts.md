---
sidebar_label: Timeouts
sidebar_position: 155
---
# Job execution timeouts

Bacalhau can limit the total time a job spends executing. If the job takes longer than the specified time to complete, it will be canceled. In this case, the job results will not be published, and the user will receive a relevant notification

By default, a Bacalhau node has 10 minutes execution timeout for all jobs which do not have their own timeout defined. Both node operators and job submitters can supply a maximum execution time
limit. If a job submitter asks for a longer execution time than permitted by a
node operator, their job will be rejected.

Applying job timeouts allows node operators to more fairly distribute the work
submitted to their nodes. It also protects users from transient errors that
results in their jobs waiting indefinitely.

## Configuring execution time limits for a job

Job submitters can pass the `--timeout` flag to any Bacalhau job submission CLI
to set a maximum job execution time. The supplied value should be a whole number
of seconds with no unit.

The timeout can also be added to an existing job spec by adding the [Timeout](../jobs/job-specification/timeouts.md)
property to the [Spec](../jobs/job-specification/job.md).

## Configuring execution time limits for a node

Node operators can pass the `--max-job-execution-timeout` flag to `bacalhau serve` to
configure the maximum job time limit. The supplied value should be a numeric
value followed by a time unit (one of `s` for seconds, `m` for minutes or `h`
for hours).

Node operators can also use configuration properties to configure execution
limits.

Compute nodes will use the properties:

| Config property | Meaning | Default value |
|---|---|---|
| `Node.Compute.JobTimeouts.MinJobExecutionTimeout` | The minimum acceptable value for a job timeout. A job will only be accepted if it is submitted with a timeout of longer than this value. | `0.5s` |
| `Node.Compute.JobTimeouts.MaxJobExecutionTimeout` | The maximum acceptable value for a job timeout. A job will only be accepted if it is submitted with a timeout of shorter than this value. |`2562047h`|
| `Node.Compute.JobTimeouts.DefaultJobExecutionTimeout` | The job timeout that will be applied to jobs that are submitted without a timeout value.  |`10m`|
|Node.Compute.JobTimeouts.JobExecutionTimeoutClientIdBypassList|The list of clients who are allowed to bypass the job execution timeout|Empty|
|Node.Compute.JobTimeouts.JobNegotiationTimeout|The minimum execution timeout supported by current node. Jobs with lower timeout requirements will not be bid on|`3m`|

Requester nodes will use the properties:

| Config property | Meaning |
|---|---|
| `Node.Requester.Timeouts.MinJobExecutionTimeout` | If a job is submitted with a timeout less than this value, the default job execution timeout will be used instead. |
| `Node.Requester.Timeouts.DefaultJobExecutionTimeout` | The timeout to use in the job if a timeout is missing or too small. |
