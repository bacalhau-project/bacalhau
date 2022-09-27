---
sidebar_label: 'CLI Reference'
sidebar_position: 9
---

# CLI Commands

:::info

The following commands refer to bacalhau cli version `v0.2.3`.
For installing or upgrading a client follow the instructions in the [installation page](./getting-started/installation.md).
Run `bacalhau version` in a terminal to check what version you have.

:::

```
❯ bacalhau --help
Compute over data

Usage:
  bacalhau [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  create      Create a job using a json or yaml file.
  describe    Describe a job on the network
  devstack    Start a cluster of bacalhau nodes for testing and development
  docker      Run a docker job on the network (see run subcommand)
  get         Get the results of a job
  help        Help about any command
  list        List jobs on the network
  run         Run a job on the network (see subcommands for supported flavors)
  serve       Start the bacalhau compute node
  version     Get the client and server version.

Flags:
      --api-host string   The host for the client and server to communicate on (via REST). Ignored if BACALHAU_API_HOST environment variable is set. (default "bootstrap.production.bacalhau.org")
      --api-port int      The port for the client and server to communicate on (via REST). Ignored if BACALHAU_API_PORT environment variable is set. (default 1234)
  -h, --help              help for bacalhau

Use "bacalhau [command] --help" for more information about a command.
```



## Create

Submit a job to the network in a declarative way by writing a jobspec instead of writing a command. 
JSON and YAML formats are accepted.

```
Create a job from a file or from stdin.

 JSON and YAML formats are accepted.

Usage:
  bacalhau create [flags]

Flags:
  -c, --concurrency int             How many nodes should run the job (default 1)
      --confidence int              The minimum number of nodes that must agree on a verification result
      --download                    Download the results and print stdout once the job has completed (implies --wait).
      --download-timeout-secs int   Timeout duration for IPFS downloads. (default 10)
  -g, --gettimeout int              Timeout for getting the results of a job in --wait (default 10)
  -h, --help                        help for create
      --ipfs-swarm-addrs string     Comma-separated list of IPFS nodes to connect to.
      --local                       Run the job locally. Docker is required
      --output-dir string           Directory to write the output to. (default ".")
      --wait                        Wait for the job to finish.
      --wait-timeout-secs int       When using --wait, how many seconds to wait for the job to complete before giving up. (default 600)
```

#### Examples

```
# Create a job using the data in job.json
bacalhau create ./job.json

# Create a job based on the JSON passed into stdin
cat job.json | job create -
```

An example jobspec in YAML format:

```yaml
apiVersion: v1alpha1
engine: Docker
verifier: Ipfs
job_spec_docker:
  image: gromacs/gromacs
  entrypoint:
    - /bin/bash
    - -c
    - echo 15 | gmx pdb2gmx -f input/1AKI.pdb -o output/1AKI_processed.gro -water spc
  env: []
job_spec_language:
  language: ''
  language_version: ''
  deterministic: false
  context:
    engine: ''
    name: ''
    cid: ''
    path: ''
  command: ''
  program_path: ''
  requirements_path: ''
resources:
  cpu: ''
  gpu: ''
  memory: ''
  disk: ''
inputs:
  - engine: ipfs
    name: ''
    cid: QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9
    path: /input
  - engine_name: urldownload
    name: ''
    url: https://foo.bar.io/foo_data.txt
    path: /app/foo_data_1.txt
outputs:
  - engine: ipfs
    name: output
    cid: ''
    path: /output
annotations: null
```

An example jobspoec in JSON format:

```json
{
  "apiVersion": "v1alpha1",
  "engine": "Docker",
  "verifier": "Ipfs",
  "job_spec_docker": {
      "image": "gromacs/gromacs",
      "entrypoint": [
          "/bin/bash",
          "-c",
          "echo 15 | gmx pdb2gmx -f input/1AKI.pdb -o output/1AKI_processed.gro -water spc"
      ],
      "env": []
  },
  "job_spec_language": {
      "language": "",
      "language_version": "",
      "deterministic": false,
      "context": {
          "engine": "",
          "name": "",
          "cid": "",
          "path": ""
      },
      "command": "",
      "program_path": "",
      "requirements_path": ""
  },
  "resources": {
      "cpu": "",
      "gpu":"",
      "memory": "",
      "disk": ""
  },
  "inputs": [
      {
          "engine": "ipfs",
          "name": "",
          "cid": "QmeeEB1YMrG6K8z43VdsdoYmQV46gAPQCHotZs9pwusCm9",
          "path": "/input"
      }
  ],
  "outputs": [
      {
          "engine": "ipfs",
          "name": "output",
          "cid": "",
          "path": "/output"
      }
  ],
  
  "annotations": null
}
```

## Describe

```
Full description of a job, in yaml format. Use 'bacalhau list' to get a list of all ids. Short form and long form of the job id are accepted.

Usage:
  bacalhau describe [id] [flags]
```

#### Example

```
# Describe a job with the full ID
bacalhau describe e3f8c209-d683-4a41-b840-f09b88d087b9

# Describe a job with the a shortened ID
bacalhau describe 47805f5c
```
## Docker run

```
Runs a job using the Docker executor on the node.

Usage:
  bacalhau docker run [flags]

Flags:
  -c, --concurrency int                How many nodes should run the job (default 1)
      --confidence int                 The minimum number of nodes that must agree on a verification result
      --cpu string                     Job CPU cores (e.g. 500m, 2, 8).
      --download                       Download the results and print stdout once the job has completed (implies --wait).
      --download-timeout-secs int      Timeout duration for IPFS downloads. (default 10)
      --engine string                  What executor engine to use to run the job (default "docker")
  -e, --env strings                    The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)
  -g, --gettimeout int                 Timeout for getting the results of a job in --wait (default 10)
      --gpu string                     Job GPU requirement (e.g. 1, 2, 8).
  -h, --help                           help for run
  -u, --input-urls strings             URL:path of the input data volumes downloaded from a URL source. Mounts data at 'path' (e.g. '-u http://foo.com/bar.tar.gz:/app/bar.tar.gz'
                                                mounts 'http://foo.com/bar.tar.gz' at '/app/bar.tar.gz'). URL can specify a port number (e.g. 'https://foo.com:443/bar.tar.gz:/app/bar.tar.gz')
                                                and supports HTTP and HTTPS.
  -v, --input-volumes strings          CID:path of the input data volumes, if you need to set the path of the mounted data.
  -i, --inputs strings                 CIDs to use on the job. Mounts them at '/inputs' in the execution.
      --ipfs-swarm-addrs string        Comma-separated list of IPFS nodes to connect to.
  -l, --labels strings                 List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.
      --local                          Run the job locally. Docker is required
      --memory string                  Job Memory requirement (e.g. 500Mb, 2Gb, 8Gb).
      --min-bids int                   Minimum number of bids that must be received before concurrency-many bids will be accepted (at random)
      --output-dir string              Directory to write the output to. (default ".")
  -o, --output-volumes strings         name:path of the output data volumes. 'outputs:/outputs' is always added.
      --publisher string               What publisher engine to use to publish the job results (default "estuary")
      --sharding-base-path string      Where the sharding glob pattern starts from - useful when you have multiple volumes. (default "/inputs")
      --sharding-batch-size int        Place results of the sharding glob pattern into groups of this size. (default 1)
      --sharding-glob-pattern string   Use this pattern to match files to be sharded.
      --skip-syntax-checking           Skip having 'shellchecker' verify syntax of the command
      --verifier string                What verification engine to use to run the job (default "noop")
      --wait                           Wait for the job to finish.
      --wait-timeout-secs int          When using --wait, how many seconds to wait for the job to complete before giving up. (default 600)
  -w, --workdir string                 Working directory inside the container. Overrides the working directory shipped with the image (e.g. via WORKDIR in Dockerfile).
```

#### Example

```
# Run a Docker job, using the image 'dpokidov/imagemagick', with a CID mounted at /input_images and an output volume mounted at /outputs in the container.
# All flags after the '--' are passed directly into the container for execution.
bacalhau docker run \
-v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images \
dpokidov/imagemagick:7.1.0-47-ubuntu \
-- magick mogrify -resize 100x100 -quality 100 -path /outputs '/input_images/*.jpg'
```

## Get

```
Get the results of the job, including stdout and stderr.

Usage:
  bacalhau get [id] [flags]

Flags:
      --download-timeout-secs int   Timeout duration for IPFS downloads. (default 600)
  -h, --help                        help for get
      --ipfs-swarm-addrs string     Comma-separated list of IPFS nodes to connect to.
      --output-dir string           Directory to write the output to. (default ".")
```

#### Example

```
# Get the results of a job.
bacalhau get 51225160-807e-48b8-88c9-28311c7899e1

# Get the results of a job, with a short ID.
bacalhau get ebd9bf2f
```

## List

```
List jobs on the network.

Usage:
  bacalhau list [flags]

Flags:
      --all                Fetch all jobs from the network (default is to filter those belonging to the user). This option may take a long time to return, please use with caution.
  -h, --help               help for list
      --hide-header        do not print the column headers.
      --id-filter string   filter by Job List to IDs matching substring.
      --no-style           remove all styling from table output.
  -n, --number int         print the first NUM jobs instead of the first 10. (default 10)
      --output string      The output format for the list of jobs (json or text) (default "text")
      --reverse            reverse order of table - for time sorting, this will be newest first. Use '--reverse=false' to sort oldest first (single quotes are required). (default true)
      --sort-by Column     sort by field, defaults to creation time, with newest first [Allowed "id", "created_at"]. (default created_at)
      --wide               Print full values in the table results
```

#### Example

```
# List jobs on the network
bacalhau list

# List jobs and output as json
bacalhau list --output json
```

## Run Python

```
Runs a job by compiling language file to WASM on the node.

Usage:
  bacalhau run python [flags]

Flags:
  -c, --command string           Program passed in as string (like python)
      --concurrency int          How many nodes should run the job (default 1)
      --confidence int           The minimum number of nodes that must agree on a verification result
      --context-path string      Path to context (e.g. python code) to send to server (via public IPFS network) for execution (max 10MiB). Set to empty string to disable (default ".")
      --deterministic            Enforce determinism: run job in a single-threaded wasm runtime with no sources of entropy. NB: this will make the python runtime executein an environment where only some librarie are supported, see https://pyodide.org/en/stable/usage/packages-in-pyodide.html (default true)
  -e, --env strings              The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)
  -h, --help                     help for python
  -v, --input-volumes strings    CID:path of the input data volumes
  -i, --inputs strings           CIDs to use on the job. Mounts them at '/inputs' in the execution.
  -l, --labels strings           List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.
  -o, --output-volumes strings   name:path of the output data volumes
  -r, --requirement string       Install from the given requirements file. (like pip)
      --verifier string          What verification engine to use to run the job (default "ipfs")
```


## Serve

```
Start the bacalhau campute node.

Usage:
  bacalhau serve [flags]

Flags:
      --estuary-api-key string               The API key used when using the estuary API.
      --filecoin-unsealed-path string        The go template that can turn a filecoin CID into a local filepath with the unsealed data.
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
      --swarm-port int                       The port to listen on for swarm connections. (default 1235)
```
