---
sidebar_label: 'Job Selection Policy'
sidebar_position: 2
---

# Job selection policy

When running a node, you can choose which jobs you want to run by using
configuration options, environment variables or flags to specify a job selection
policy.

| Config property | `serve` flag | Default value | Meaning |
|---|---|---|---|
| Node.Compute.JobSelection.Locality | `--job-selection-data-locality` | Anywhere | Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere"). |
| Node.Compute.JobSelection.ProbeExec | `--job-selection-probe-exec` | unused | Use the result of an external program to decide if we should take on the job. |
| Node.Compute.JobSelection.ProbeHttp | `--job-selection-probe-http` | unused | Use the result of a HTTP POST to decide if we should take on the job. |
| Node.Compute.JobSelection.RejectStatelessJobs | `--job-selection-reject-stateless` | False | Reject jobs that don't specify any [input data](../data-ingestion/index.md). |
| Node.Compute.JobSelection.AcceptNetworkedJobs | `--job-selection-accept-networked` | False | Accept jobs that require [network connections](../networking-instructions/networking.md). |


## Job selection probes

If you want more control over the decision-making process for accepting or rejecting jobs, you can use the `--job-selection-probe-exec` and `--job-selection-probe-http` flags when starting a node.

These are external programs that are passed the following data structure so that they can make a decision about whether or not to take on a job:

```go
type BidStrategyRequest struct {
	NodeID   string
	Job      models.Job
	Callback *url.URL
}
```

## Exec probe
The `exec` probe is a script to run that will be given the job data on `stdin`, and must exit with status code 0 if the job should be run. Make sure that your node has all the necessary dependencies for the script to work, if any. Command example:
```bash
bacalhau serve --job-selection-probe-exec my_script.sh
```
## HTTP probe
The `http` probe is a URL where the job data will be sent as a POST request. Make sure your node has the network access needed to access the specified URL. Command example:
```bash
bacalhau serve --job-selection-probe-http http://path.to.your.resource
```

The decision to accept or reject a job is made based on the response code and the body of the response. The job will be rejected if the answer code is non-positive (4xx, 5xx). If the answer code is positive, the answer body is checked. It is ignored if it is not a JSON Blob. Otherwise, it should match the [following
schema](https://github.com/bacalhau-project/bacalhau/blob/885d53e93b01fb343294d7ddbdbffe89918db800/pkg/bidstrategy/type.go#L18-L22)
and the decision will be made on the basis of its contents.

```json
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "type": "object",
  "properties": {
    "shouldBid": {
      "description": "If the job should be accepted",
      "type": "boolean"
    },
    "shouldWait": {
      "description": "If the node should wait for an async response that will come later. `shouldBid` will be ignored",
      "type": "boolean",
      "default": false,
    },
    "reason": {
      "description": "Human-readable string explaining why the job should be accepted or rejected, or why the wait is required",
      "type": "string"
    }
  },
  "required": [
    "shouldBid",
    "reason"
  ]
}
```

For example, the following response will reject the job:

```json
{
  "shouldBid": false,
  "reason": "The job did not pass this specific validation: ...",
}
```
