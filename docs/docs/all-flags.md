---
sidebar_label: "CLI Reference"
sidebar_position: 7
---

# CLI Commands

:::info

The following commands refer to bacalhau cli version `v1.0.3`.
For installing or upgrading a client, follow the instructions in the [installation page](https://docs.bacalhau.org/getting-started/installation).
Run `bacalhau version` in a terminal to check what version you have.

:::

```
❯ bacalhau --help
Compute over data

Usage:
  bacalhau [command]

Available Commands:
  cancel      Cancel a previously submitted job
  completion  Generate the autocompletion script for the specified shell
  create      Create a job using a json or yaml file.
  describe    Describe a job on the network
  devstack    Start a cluster of bacalhau nodes for testing and development
  docker      Run a docker job on the network (see run subcommand)
  get         Get the results of a job
  help        Help about any command
  id          Show bacalhau node id info
  list        List jobs on the network
  logs        Follow logs from a currently executing job
  run         Run a job on the network (see subcommands for supported flavors)
  serve       Start the bacalhau compute node
  validate    validate a job using a json or yaml file.
  version     Get the client and server version.

Flags:
      --api-host string   The host for the client and server to communicate on (via REST). Ignored if BACALHAU_API_HOST environment variable is set. (default "bootstrap.production.bacalhau.org")
      --api-port int      The port for the client and server to communicate on (via REST). Ignored if BACALHAU_API_PORT environment variable is set. (default 1234)
  -h, --help              help for bacalhau

Use "bacalhau [command] --help" for more information about a command.
```

## Cancel

Cancels a job that was previously submitted and stops it running if it has not yet completed.

```
Cancel a previously submitted job.

Usage:
  ./bin/darwin_arm64/bacalhau cancel [id] [flags]

Flags:
  -h, --help    help for cancel
      --quiet   Do not print anything to stdout or stderr
```

#### Examples

```
Examples:
  # Cancel a previously submitted job
  bacalhau cancel 51225160-807e-48b8-88c9-28311c7899e1

  # Cancel a job, with a short ID.
  bacalhau cancel ebd9bf2f
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
      --download                    Download the results and print stdout once the job has completed (implies --wait).
      --download-timeout-secs int   Timeout duration for IPFS downloads. (default 10)
  -g, --gettimeout int              Timeout for getting the results of a job in --wait (default 10)
  -h, --help                        help for create
      --ipfs-swarm-addrs string     Comma-separated list of IPFS nodes to connect to.
      --local                       Run the job locally. Docker is required
      --output-dir string           Directory to write the output to. (default ".")
      --wait                        Wait for the job to finish. Use --wait=false to not wait.
      --wait-timeout-secs int       When using --wait, how many seconds to wait for the job to complete before giving up. (default 600)
```

#### Examples

```
Examples:
  # Create a job using the data in job.yaml
  bacalhau create ./job.yaml

  # Create a new job from an already executed job
  bacalhau describe 6e51df50 | bacalhau create -
```

An example job in YAML format:

```yaml
spec:
  engine: Docker
  verifier: Noop
  publisher: IPFS
  docker:
    image: ubuntu
    entryPoint:
      - echo
    parameters:
      - Hello
      - World
  outputs:
    - name: outputs
      path: /outputs
deal:
  concurrency: 1
```

### UCAN Invocation format

You can also specify a job to run using a [UCAN Invocation](https://github.com/ucan-wg/invocation) object in JSON format. For the fields supported by Bacalhau, see the [IPLD schema](https://github.com/bacalhau-project/bacalhau/blob/main/pkg/model/schemas/bacalhau.ipldsch).

There is no support for sharding, concurrency or minimum bidding for these jobs.

#### Examples

Refers to example models at bacalhau repository under [pkg/model/tasks](https://github.com/bacalhau-project/bacalhau/tree/main/pkg/model/tasks)

An example UCAN Invocation that runs the same job as the above example would look like:

```json
{
  "with": "ubuntu",
  "do": "docker/run",
  "inputs": {
    "entrypoint": ["echo"],
    "parameters": ["hello", "world"],
    "workdir": "/",
    "mounts": {},
    "outputs": {
      "/outputs": ""
    }
  },
  "meta": {
    "bacalhau/config": {
      "verifier": 1,
      "publisher": 4,
      "annotations": ["hello"],
      "resources": {
        "cpu": 1,
        "disk": 1073741824,
        "memory": 1073741824,
        "gpu": 0
      },
      "timeout": 300e9,
      "dnt": false
    }
  }
}
```

An example UCAN Invocation that runs a WebAssembly job might look like:

```json
{
	"with": "ipfs://bafybeig7mdkzcgpacpozamv7yhhaelztfrnb6ozsupqqh7e5uyqdkijegi",
	"do": "wasm32-wasi/run",
	"inputs": {
		"entrypoint": "_start",
		"parameters": ["/inputs/data.tar.gz"],
		"mounts": {
			"/inputs": "https://www.example.com/data.tar.gz"
		},
		"outputs": {
			"/outputs": ""
		},
		"env": {
			"HELLO": "world"
		}
	},
	"meta": {
    }
  }
}
```

## Describe

```
Full description of a job, in yaml format. Use 'bacalhau list' to get a list of all ids. Short form and long form of the job id are accepted.

Usage:
  bacalhau describe [id] [flags]

Flags:
  -h, --help             help for describe
      --include-events   Include events in the description (could be noisy)
      --spec             Output Jobspec to stdout
```

#### Example

```
Examples:
  # Describe a job with the full ID
  bacalhau describe e3f8c209-d683-4a41-b840-f09b88d087b9

  # Describe a job with the a shortened ID
  bacalhau describe 47805f5c

  # Describe a job and include all server and local events
  bacalhau describe --include-events b6ad164a
```

## Docker run

```
Runs a job using the Docker executor on the node.

Usage:
  bacalhau docker run [flags] IMAGE[:TAG|@DIGEST] [COMMAND] [ARG...]

Examples:
  # Run a Docker job, using the image 'dpokidov/imagemagick', with a CID mounted at /input_images and an output volume mounted at /outputs in the container. All flags after the '--' are passed directly into the container for execution.
  bacalhau docker run \
  -i src=ipfs://QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72,dst=/input_images \
  dpokidov/imagemagick:7.1.0-47-ubuntu \
  -- magick mogrify -resize 100x100 -quality 100 -path /outputs '/input_images/*.jpg'

  # Dry Run: check the job specification before submitting it to the bacalhau network
  bacalhau docker run --dry-run ubuntu echo hello

  # Save the job specification to a YAML file
  bacalhau docker run --dry-run ubuntu echo hello > job.yaml

  # Specify an image tag (default is 'latest' - using a specific tag other than 'latest' is recommended for reproducibility)
  bacalhau docker run ubuntu:bionic echo hello

  # Specify an image digest
  bacalhau docker run ubuntu@sha256:35b4f89ec2ee42e7e12db3d107fe6a487137650a2af379bbd49165a1494246ea echo hello

Flags:
  -c, --concurrency int                  How many nodes should run the job (default 1)
      --confidence int                   The minimum number of nodes that must agree on a verification result
      --cpu string                       Job CPU cores (e.g. 500m, 2, 8).
      --domain stringArray               Domain(s) that the job needs to access (for HTTP networking)
      --download                         Should we download the results once the job is complete?
      --download-timeout-secs duration   Timeout duration for IPFS downloads. (default 5m0s)
      --dry-run                          Do not submit the job, but instead print out what will be submitted
      --engine string                    What executor engine to use to run the job (default "docker")
  -e, --env strings                      The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)
      --filplus                          Mark the job as a candidate for moderation for FIL+ rewards.
  -f, --follow                           When specified will follow the output from the job as it runs
  -g, --gettimeout int                   Timeout for getting the results of a job in --wait (default 10)
      --gpu string                       Job GPU requirement (e.g. 1, 2, 8).
  -h, --help                             help for run
      --id-only                          Print out only the Job ID on successful submission.
  -i, --input storage                    Mount URIs as inputs to the job. Can be specified multiple times. Format: src=URI,dst=PATH[,opt=key=value]
                                         Examples:
                                         # Mount IPFS CID to /inputs directory
                                         -i ipfs://QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72

                                         # Mount S3 object to a specific path
                                         -i s3://bucket/key,dst=/my/input/path

                                         # Mount S3 object with specific endpoint and region
                                         -i src=s3://bucket/key,dst=/my/input/path,opt=endpoint=https://s3.example.com,opt=region=us-east-1

      --ipfs-swarm-addrs string          Comma-separated list of IPFS nodes to connect to. (default "/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL,/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF,/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3")
  -l, --labels strings                   List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.
      --local                            Run the job locally. Docker is required
      --memory string                    Job Memory requirement (e.g. 500Mb, 2Gb, 8Gb).
      --min-bids int                     Minimum number of bids that must be received before concurrency-many bids will be accepted (at random)
      --network network-type             Networking capability required by the job (default "nats")
      --node-details                     Print out details of all nodes (overridden by --id-only).
      --output-dir string                Directory to write the output to.
  -o, --output-volumes strings           name:path of the output data volumes. 'outputs:/outputs' is always added.
  -p, --publisher publisher              Where to publish the result of the job (default IPFS)
      --raw                              Download raw result CIDs instead of merging multiple CIDs into a single result
  -s, --selector string                  Selector (label query) to filter nodes on which this job can be executed, supports '=', '==', and '!='.(e.g. -s key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.
      --skip-syntax-checking             Skip having 'shellchecker' verify syntax of the command
      --timeout float                    Job execution timeout in seconds (e.g. 300 for 5 minutes and 0.1 for 100ms) (default 1800)
      --verifier string                  What verification engine to use to run the job (default "noop")
      --wait                             Wait for the job to finish. (default true)
      --wait-timeout-secs int            When using --wait, how many seconds to wait for the job to complete before giving up. (default 600)
  -w, --workdir string                   Working directory inside the container. Overrides the working directory shipped with the image (e.g. via WORKDIR in Dockerfile).
```

## Get

```
Get the results of the job, including `stdout` and `stderr`.

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

## Logs

Retrieves the log output (stdout, and stderr) from a job.
If the job is still running it is possible to follow the logs after the previously generated logs are retrieved.

```
Follow logs from a currently executing job

Usage:
  ./bin/darwin_arm64/bacalhau logs [flags] [id]

Flags:
  -f, --follow   Follow the logs in real-time after retrieving the current logs.
  -h, --help     help for logs
```

#### Examples

```
Examples:
  # Follow logs for a previously submitted job
  bacalhau logs -f 51225160-807e-48b8-88c9-28311c7899e1

  # Retrieve the log output with a short ID, but don't follow any newly generated logs
  bacalhau logs ebd9bf2f
```

## Run Python

```
Runs a job by compiling language file to WASM on the node.

Usage:
  bacalhau run python [flags]

Examples:
  # Run a simple "Hello, World" script within the current directory
  bacalhau run python -- hello-world.py

Flags:
  -c, --command string                   Program passed in as string (like python)
      --concurrency int                  How many nodes should run the job (default 1)
      --confidence int                   The minimum number of nodes that must agree on a verification result
      --context-path string              Path to context (e.g. python code) to send to server (via public IPFS network) for execution (max 10MiB). Set to empty string to disable (default ".")
      --deterministic                    Enforce determinism: run job in a single-threaded wasm runtime with no sources of entropy. NB: this will make the python runtime execute in an environment where only some libraries are supported, see https://pyodide.org/en/stable/usage/packages-in-pyodide.html (default true)
      --download                         Should we download the results once the job is complete?
      --download-timeout-secs duration   Timeout duration for IPFS downloads. (default 5m0s)
  -e, --env strings                      The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)
  -f, --follow                           When specified will follow the output from the job as it runs
  -g, --gettimeout int                   Timeout for getting the results of a job in --wait (default 10)
  -h, --help                             help for python
      --id-only                          Print out only the Job ID on successful submission.
  -i, --input storage                    Mount URIs as inputs to the job. Can be specified multiple times. Format: src=URI,dst=PATH[,opt=key=value]
                                         Examples:
                                         # Mount IPFS CID to /inputs directory
                                         -i ipfs://QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72

                                         # Mount S3 object to a specific path
                                         -i s3://bucket/key,dst=/my/input/path

                                         # Mount S3 object with specific endpoint and region
                                         -i src=s3://bucket/key,dst=/my/input/path,opt=endpoint=https://s3.example.com,opt=region=us-east-1

      --ipfs-swarm-addrs string          Comma-separated list of IPFS nodes to connect to. (default "/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL,/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF,/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3")
  -l, --labels strings                   List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.
      --local                            Run the job locally. Docker is required
      --min-bids int                     Minimum number of bids that must be received before concurrency-many bids will be accepted (at random)
      --node-details                     Print out details of all nodes (overridden by --id-only).
      --output-dir string                Directory to write the output to.
  -o, --output-volumes strings           name:path of the output data volumes
      --raw                              Download raw result CIDs instead of merging multiple CIDs into a single result
  -r, --requirement string               Install from the given requirements file. (like pip)
      --timeout float                    Job execution timeout in seconds (e.g. 300 for 5 minutes and 0.1 for 100ms) (default 1800)
      --wait                             Wait for the job to finish. (default true)
      --wait-timeout-secs int            When using --wait, how many seconds to wait for the job to complete before giving up. (default 600)
```

## Serve

```
Start a bacalhau node.

Usage:
  bacalhau serve [flags]

Examples:
  # Start a private bacalhau requester node
  bacalhau serve
  # or
  bacalhau serve --node-type requester

  # Start a private bacalhau hybrid node that acts as both compute and requester
  bacalhau serve --node-type compute --node-type requester
  # or
  bacalhau serve --node-type compute,requester

  # Start a private bacalhau node with a persistent local IPFS node
  BACALHAU_SERVE_IPFS_PATH=/data/ipfs bacalhau serve

  # Start a public bacalhau requester node
  bacalhau serve --peer env --private-internal-ipfs=false

Flags:
      --filecoin-unsealed-path string                    The go template that can turn a filecoin CID into a local filepath with the unsealed data.
  -h, --help                                             help for serve
      --host string                                      The host to listen on (for both api and swarm connections). (default "0.0.0.0")
      --ipfs-connect string                              The ipfs host multiaddress to connect to, otherwise an in-process IPFS node will be created if not set.
      --ipfs-swarm-addr strings                          IPFS multiaddress to connect the in-process IPFS node to - cannot be used with --ipfs-connect.
      --job-execution-timeout-bypass-client-id strings   List of IDs of clients that are allowed to bypass the job execution timeout check
      --job-selection-accept-networked                   Accept jobs that require network access.
      --job-selection-data-locality string               Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere"). (default "local")
      --job-selection-probe-exec string                  Use the result of a exec an external program to decide if we should take on the job.
      --job-selection-probe-http string                  Use the result of a HTTP POST to decide if we should take on the job.
      --job-selection-reject-stateless                   Reject jobs that don't specify any data.
      --labels stringToString                            Labels to be associated with the node that can be used for node selection and filtering. (e.g. --labels key1=value1,key2=value2) (default [])
      --limit-job-cpu string                             Job CPU core limit for single job (e.g. 500m, 2, 8).
      --limit-job-gpu string                             Job GPU limit for single job (e.g. 1, 2, or 8).
      --limit-job-memory string                          Job Memory limit for single job  (e.g. 500Mb, 2Gb, 8Gb).
      --limit-total-cpu string                           Total CPU core limit to run all jobs (e.g. 500m, 2, 8).
      --limit-total-gpu string                           Total GPU limit to run all jobs (e.g. 1, 2, or 8).
      --limit-total-memory string                        Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).
      --lotus-max-ping duration                          The highest ping a Filecoin miner could have when selecting. (default 2s)
      --lotus-path-directory string                      Location of the Lotus Filecoin configuration directory.
      --lotus-storage-duration duration                  Duration to store data in Lotus Filecoin for.
      --lotus-upload-directory string                    Directory to use when uploading content to Lotus Filecoin.
      --node-type strings                                Whether the node is a compute, requester or both. (default [requester])
      --peer string                                      A comma-separated list of libp2p multiaddress to connect to. Use "none" to avoid connecting to any peer, "env" to connect to the default peer list of your active environment (see BACALHAU_ENVIRONMENT env var). (default "none")
      --private-internal-ipfs                            Whether the in-process IPFS node should auto-discover other nodes, including the public IPFS network - cannot be used with --ipfs-connect. Use "--private-internal-ipfs=false" to disable. To persist a local Ipfs node, set BACALHAU_SERVE_IPFS_PATH to a valid path. (default true)
      --swarm-port int                                   The port to listen on for swarm connections. (default 1235)

Global Flags:
      --api-host string         The host for the client and server to communicate on (via REST).
                                Ignored if BACALHAU_API_HOST environment variable is set. (default "bootstrap.production.bacalhau.org")
      --api-port uint16         The port for the client and server to communicate on (via REST).
                                Ignored if BACALHAU_API_PORT environment variable is set. (default 1234)
      --log-mode logging-mode   Log format: 'default','station','json','combined','event' (default default)
```
