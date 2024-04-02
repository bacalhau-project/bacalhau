# Debugging Locally

Useful tricks for debugging bacalhau when developing it locally.

## Logging

We use the [zerolog](https://github.com/rs/zerolog) library for logging and so the usual log levels are supported (`debug`, `info`, `warn`, `error`, `fatal`).

The log level is controlled by the `LOG_LEVEL` variable:

```bash
LOG_LEVEL=debug go run . devstack
```

An example of printing a log at a certain level (this is literally just using the zero log library):

```go
log.Debug().Msgf("Bid accepted: Server (id: %s) - Job (id: %s)", nodeID, job.Id)
```

We also have the `LOG_TYPE` variable which controls what format the log messages are printed in:

 * `text` (default): Prints the log message using `zerolog.ConsoleWriter` so you see text output.
 * `json`: Prints line delimited JSON logs
 * `event`: Prints only the event logs
 * `combined`: Prints text, json and event logs

## Event log

Event logs are useful when you need to understand the flow of events through the system.

They are much less noisy and are only called from the requester node and compute node when a job is transitioned to a new state.

To print only event logs - you use the `LOG_TYPE` variable:

```bash
cd pkg/test/devstack
LOG_TYPE=event go test -v -run ^TestCatFileStdout$ -count 1 .
```

It's sometimes useful to see the text output on stdout but also write just the event log to a file - for this the `LOG_EVENT_FILE` variable can be used:

```bash
LOG_TYPE=text LOG_EVENT_FILE=/tmp/bacalhau_events.json go test -v -run ^TestCatFileStdout$ -count 1 .
```

An example of calling the event log library:

```go
logger.LogJobEvent(logger.JobEvent{
  Node: nodeID,
  Type: "compute_node:run",
  Job:  job.Id,
  Data: job,
})
```
