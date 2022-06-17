---
sidebar_label: 'CLI Flags'
sidebar_position: 1
---

# CLI Commands

```bash
  devstack    Start a cluster of bacalhau nodes for testing and development
  help        Help about any command
  list        List jobs on the network
  run         Run a job on the network
  serve       Start the bacalhau compute node
```

## Top level flags

```bash
      --api-host string   The host for the client and server to communicate on (via REST). (default "bootstrap.production.bacalhau.org")
      --api-port int      The port for the client and server to communicate on (via REST). (default 1234)
  -h, --help              help for bacalhau
```

### List

```bash
  -h, --help               help for list
      --hide-header        do not print the column headers.
      --id-filter string   filter by Job List to IDs matching substring.
      --no-style           remove all styling from table output.
  -n, --number int         print the first NUM jobs instead of the first 10. (default 10)
      --output string      The output format for the list of jobs (json or text) (default "text")
      --reverse            reverse order of table - for time sorting, this will be newest first.
      --sort-by Column     sort by field, defaults to creation time, with newest first [Allowed "id", "created_at"].
      --wide               Print full values in the table results
```

### Serve

```bash
  -h, --help                                 help for serve
      --host string                          The host to listen on (for both api and swarm connections). (default "0.0.0.0")
      --ipfs-connect string                  The ipfs host multiaddress to connect to.
      --job-selection-data-locality string   Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere"). (default "local")
      --job-selection-probe-exec string      Use the result of a exec an external program to decide if we should take on the job.
      --job-selection-probe-http string      Use the result of a HTTP POST to decide if we should take on the job.
      --job-selection-reject-stateless       Reject jobs that don't specify any data.
      --peer string                          The libp2p multiaddress to connect to.
      --port int                             The port to listen on for swarm connections. (default 1235)
```
