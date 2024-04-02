---
sidebar_label: 'Custom Job Types'
sidebar_position: 1
title: 'Custom Job Types'
description: Submitting jobs that are not docker or wasm jobs.
---

With the addition of the experimental `exec` command to the bacalhau command-line interface (CLI),
it is possible to submit jobs that are not docker or wasm jobs.
A job that is submitted as a custom job type is translated by the requester node
into one of the supported execution environments, Docker or WebAssembly.

For each new job type, support is required in the client (either CLI or SDK),
and the requester node that receives the job submission.

:::info
As an experimental feature, the `exec` command is added to examine how the
feature can be used, extended and improved. It is possible that this command
may be removed in future versions if an alternative approach is discovered to
be more effective.
:::


## CLI support

Within the CLI each custom job type requires a template file,
a JSON representation of the job with a .tpl extension,
found in the [templates folder](https://github.com/bacalhau-project/bacalhau/tree/main/cmd/cli/exec/templates). This template defines the base components of the job and is extended by the command line parameters provided to exec.

In addition to the usual runtime and specification flags, that can be found in [the CLI reference for exec](/dev/cli-reference/cli/job/exec/), the `--code` parameter allows for single code files, or directories of code files to be added to the specification.  By default they will be added inline to the job specification, although the requester node may chose to change the storage provider for the code. There is however a hard-limit of 10MB for the attached code.


## Requester node

When a job is received at the requester node, it determines how to process the job based
on the engine type given in the job specification.  For engine types that are docker or
webassembly, they are processed immediately.

Other engine types are processed by the [translation package](https://github.com/bacalhau-project/bacalhau/blob/main/pkg/translation/translation.go) and tasks within the job translated by the server-side job translators, where one is provided for each custom job type.  These [translators](https://github.com/bacalhau-project/bacalhau/blob/main/pkg/translation/translation.go#L34C1-L38C2) are responsible for updating the task so that it is able to run on either the docker or webassembly runtimes.

Once a job has been translated, for example: Python -> Docker then the new job ID is returned to the client as if it were a normal job. The newly translated job may have a dependency on an external container which performs extra-steps during deployment, it is up to the translator to ensure it specifies the correct container or wasm module.


## Initial job types

The initial implementation of custom job types includes job types for Python and DuckDB. Both of these custom job types support the `--code` parameter to simplify the code being sent to  the requester node - for python this is likely python code, for duckdb it is expected to be sql.  Command line parameters that follow the job type are used as the parameters to the translator, so `python /code/app.py /inputs/data.txt` will result in parameters that are `['python', '/code/app.py', '/inputs/data.txt']`.  Any flags that need to be passed to the python process, for instance should app.py require -f for the final parameter, must be sent after the bash -- separator.

### Dependencies

For Python, there is a likelihood that the code to be executed has dependencies, and these can be specified one of several ways.  The docker image for the Python executor has a specific entrypoint that attempts to determine and then install the requirements.

In a single file attachment (via --code) any pip commands in the module documentation are executed before the code, for instance the following code will install colorama before interpreting the script:

```python
"""
pip install colorama
"""

from colorama import Fore
```

If a `pyproject.toml` is found in a directory specified by `--code`, then it is used to install the dependencies.  If the file has a poetry section, then `poetry install` is used instead.

A `requirements.txt` file will be used with `pip install -r` to install requirements, if found in the specified directory.

Failing everything else, if a setup.py is present it will be used with `python -m pip install -e .`.

To aid in debugging, the output of the dependency resolution is written to a file called `requirements.log` in any /outputs directory.

The network requirements must be specified in the job specification either by the template, or the requester translator. This will mean that the compute nodes will be required to allow network access.


## Adding a custom job type to Bacalhau

To add a new experimental custom job type to Bacalhau, it is necessary to add three components. The CLI template, the requester node Transformer, and the docker image or wasm module that will execute the job on the compute node.

### CLI changes

1. Add a template named after the command to be added to the [templates directory](https://github.com/bacalhau-project/bacalhau/tree/main/cmd/cli/exec/templates) (./cmd/cli/exec/templates). For example adding a template called `datafusion.tpl` will enable the command `exec datafusion`.
2. Ensure the template contains a single task that specifies an engine that can be processed by your translator.
3. Add any templated values you wish to be filled with full exec parameters. For example, ` "Version": "{{or (index . "version") "3.11"}}"` will add the value from `--version` or default to 3.11

### Requester translator

1. Add a new implementation of [Translator](https://github.com/bacalhau-project/bacalhau/blob/main/pkg/translation/translation.go#L30-L38) to the [translators directory](https://github.com/bacalhau-project/bacalhau/tree/main/pkg/translation/translators) in the translation package.
2. Ensure the translation replaces the engine in the task it is provided with a new engine that is either a valid docker or webassembly engine.
3. Add your translator to the [standard translations provider](https://github.com/bacalhau-project/bacalhau/blob/main/pkg/translation/translation.go#L45-L52).

### Runtime

1. Create a docker image, or webassembly module that can be used in step 2 of the requester translator.
2. Ensure the docker image is both versioned, and obtainable from a public docker registry. If using webassembly, ensure the module is available either via IPFS.
