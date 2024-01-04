---
sidebar_label: Timeouts
sidebar_position: 155
---
# Job execution timeouts

Bacalhau can limit the total time a job spends executing. A job that spends too
long executing will be cancelled and no results will be published.

By default, a Bacalhau node does not enforce any limit on job execution time.
Both node operators and job submitters can supply a maximum execution time
limit. If a job submitter asks for a longer execution time than permitted by a
node operator, their job will be rejected.

Applying job timeouts allows node operators to more fairly distribute the work
submitted to their nodes. It also protects users from transient errors that
results in their jobs waiting indefinitely.

## Configuring execution time limits for a job

Job submitters can pass the `--timeout` flag to any Bacalhau job submission CLI
to set a maximum job execution time. The supplied value should be a whole number
of seconds with no unit.

The timeout can also be added to an existing job spec by adding the `Timeout`
property to the `Spec`.

## Configuring execution time limits for a node

Node operators can pass the `--max-job-execution-timeout` flag to `bacalhau serve` to
configure the maximum job time limit. The supplied value should be a numeric
value followed by a time unit (one of `s` for seconds, `m` for minutes or `h`
for hours).

Node operators can also use configuration properties to configure execution
limits.

Compute nodes will use the properties:

| Config property | Meaning |
|---|---|
| `Node.Compute.JobTimeouts.MinJobExecutionTimeout` | The minimum acceptable value for a job timeout. A job will only be accepted if it is submitted with a timeout of longer than this value. |
| `Node.Compute.JobTimeouts.MaxJobExecutionTimeout` | The maximum acceptable value for a job timeout. A job will only be accepted if it is submitted with a timeout of shorter than this value. |
| `Node.Compute.JobTimeouts.DefaultJobExecutionTimeout` | The job timeout that will be applied to jobs that are submitted without a timeout value.  |

Requester nodes will use the properties:

| Config property | Meaning |
|---|---|
| `Node.Requester.Timeouts.MinJobExecutionTimeout` | If a job is submitted with a timeout less than this value, the default job execution timeout will be used instead. |
| `Node.Requester.Timeouts.DefaultJobExecutionTimeout` | The timeout to use in the job if a timeout is missing or too small. |
