---
sidebar_label: 'Overview' sidebar_position: 100
---

The `bacalhau serve` command will start a compute node that will run jobs on the network.

## Networking

There are two main network ports used by the bacalhau server:

 * `REST API` (default port 1234)
 * `libp2p swarm` (default port 1235)

The following CLI flags control the network ports:

 * `--api-port` the port for the REST API to listen on
 * `--port` the libp2p swarm port to use

You can also control which interfaces to bind to:

 * `--api-host` the host to listen on for the REST API

## IPFS server

You will need the multiaddress of a running IPFS daemon for the bacalhau node to connect to.  It will use this IPFS server to mount storage volumes into jobs and to save the results of jobs.

You connect to the IPFS daemon by specifying the multiaddress of the daemon in the `--ipfs-connect` flag.

## Bacalhau swarm

The bacalhau swarm is a libp2p network that is used to communicate with other bacalhau nodes.

To join other nodes on the network you use the `--peer` flag - you should pass a multiaddress of the node you want to join.

## Capacity

```
  --limit-job-cpu string                 Job CPU core limit for single job (e.g. 500m, 2, 8).
  --limit-job-gpu string                 Job GPU limit for single job (e.g. 1, 2, or 8).
  --limit-job-memory string              Job Memory limit for single job  (e.g. 500Mb, 2Gb, 8Gb).
  --limit-total-cpu string               Total CPU core limit to run all jobs (e.g. 500m, 2, 8).
  --limit-total-gpu string               Total GPU limit to run all jobs (e.g. 1, 2, or 8).
  --limit-total-memory string            Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).
```

These are the flags that control the capacity of the bacalhau node and the limits for jobs that might be run.

The `--limit-total-*` flags control the total system resources you want to give to the network.  If left blank - the system will attempt to detect these values automatically.

The `--limit-job-*` flags control the maximum amount of resources a single job can consume for it to be selected for execution.

## Job selection

```
  --job-selection-data-locality string   Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere"). (default "local")
  --job-selection-probe-exec string      Use the result of a exec an external program to decide if we should take on the job.
  --job-selection-probe-http string      Use the result of a HTTP POST to decide if we should take on the job.
  --job-selection-reject-stateless       Reject jobs that don't specify any data.
```

These are the flags that control how the bacalhau node selects jobs to run.

The `--job-selection-data-locality` flag (which can be "local" or "anywhere") controls whether the data used for a job has a actually live on the IPFS server you are connected to.

The `--job-selection-reject-stateless` controls whether you want to accept jobs that don't use any data volumes.

## Job selection hooks

If you want more control over making the decision to take on jobs you can use the `--job-selection-probe-exec` and `--job-selection-probe-http` flags.

These are external programs that are passed the following data structure so they can make a decision about whether or not to take on a job:

```json
{
  "node_id": "XXX",
  "job_id": "XXX",
  "spec": {
    "engine": "docker",
    "verifier": "ipfs",
    "job_spec_vm": {
      "image": "ubuntu:latest",
      "entrypoint": ["cat", "/file.txt"]
    },
    "inputs": [{
      "engine": "ipfs",
      "cid": "XXX",
      "path": "/file.txt"
    }]
  }
}
```

The `exec` probe is a script to run that will be given the job data on `stdin` and must exit with status code 0 if the job should be run.

The `http` probe is a URL to POST the job data to and must return a 200 status code if the job should be run.