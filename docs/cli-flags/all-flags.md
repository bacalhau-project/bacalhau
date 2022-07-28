---
sidebar_label: 'CLI Flags'
sidebar_position: 1
---

# CLI Commands

```bash
  apply       Submit a job.json or job.yaml file and run it on the network
  completion  Generate the autocompletion script for the specified shell
  describe    Describe a job on the network
  devstack    Start a cluster of bacalhau nodes for testing and development
  docker      Run a docker job on the network (see run subcommand)
  get         Get the results of a job
  help        Help about any command
  list        List jobs on the network
  run         Run a job on the network (see subcommands for supported flavors)
  serve       Start the bacalhau compute node
  version     Get the client and server version.
```

## Top level flags

```bash
      --api-host string   The host for the client and server to communicate on (via REST). Ignored if BACALHAU_API_HOST environment variable is set. (default "bootstrap.production.bacalhau.org")
      --api-port int      The port for the client and server to communicate on (via REST). Ignored if BACALHAU_API_PORT environment variable is set. (default 1234)
  -h, --help              help for bacalhau
```

### Docker

```bash
  run         Run a docker job on the network
```

#### Run

```bash
  -c, --concurrency int          How many nodes should run the job (default 1)
      --cpu string               Job CPU cores (e.g. 500m, 2, 8).
      --engine string            What executor engine to use to run the job (default "docker")
  -e, --env strings              The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)
  -g, --gettimeout int           Timeout for getting the results of a job in --wait (default 10)
      --gpu string               Job GPU requirement (e.g. 1, 2, 8).
  -h, --help                     help for run
  -u, --input-urls strings       URL:path of the input data volumes downloaded from a URL source. Mounts data at 'path' (e.g. '-u http://foo.com/bar.tar.gz:/app/bar.tar.gz' mounts 'http://foo.com/bar.tar.gz' at '/app/bar.tar.gz'). URL can specify a port number (e.g. 'https://foo.com:443/bar.tar.gz:/app/bar.tar.gz') and supports HTTP and HTTPS.
  -v, --input-volumes strings    CID:path of the input data volumes, if you need to set the path of the mounted data.
  -i, --inputs strings           CIDs to use on the job. Mounts them at '/inputs' in the execution.
  -l, --labels strings           List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.
      --memory string            Job Memory requirement (e.g. 500Mb, 2Gb, 8Gb).
  -o, --output-volumes strings   name:path of the output data volumes. 'outputs:/outputs' is always added.
      --skip-syntax-checking     Skip having 'shellchecker' verify syntax of the command
      --verifier string          What verification engine to use to run the job (default "ipfs")
  -w, --wait                     Wait For Job To Finish And Print Output
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
      --limit-job-cpu string                 Job CPU core limit for single job (e.g. 500m, 2, 8).
      --limit-job-gpu string                 Job GPU limit for single job (e.g. 1, 2, or 8).
      --limit-job-memory string              Job Memory limit for single job  (e.g. 500Mb, 2Gb, 8Gb).
      --limit-total-cpu string               Total CPU core limit to run all jobs (e.g. 500m, 2, 8).
      --limit-total-gpu string               Total GPU limit to run all jobs (e.g. 1, 2, or 8).
      --limit-total-memory string            Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).
      --metrics-port int                     The port to serve prometheus metrics on. (default 2112)
      --peer string                          The libp2p multiaddress to connect to.
      --port int                             The port to listen on for swarm connections. (default 1235)
```
