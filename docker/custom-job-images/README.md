# Custom Job Images 

This directory contains docker images used by the default custom job types, duckdb and python.
These images are used in the translation layer at the orchestrator, where custom job types are 
converted into jobs for one of our supported execution environments (as of 1.2 this is docker 
and wasm).  

These images make up a bundle that makes up 'custom job types', alongside the translation layer (that converts a 'python' job to a 'docker' job), and the template available to the CLI. 

## Images 

### Python - 3.11 

`exec-python-3.11` provides a Python image with access to Python 3.11, build-essentials, and 
a default set of installed requirements.  To add more default requirements, add them to [python/base_requirements.txt](python/base_requirements.txt). 

The image expects a tgz to be mounted at /code from where there build/launcher.py process will:

* Extract it
* Determine requirements method 
* Install requirements 
* Execute the command provided by the user 

If an /outputs folder exists, the stdout/stderr from the requirements installation process is written to /outputs/requirements.log for debugging.

### DuckDB

`exec-duckdb` provides an installation of duckdb installed in the image root folder.  With appropriately mounted inputs, the user is able to specify all of the required parameters for running duckdb tasks (e.g. -csv -c "query")

## Building 

Each image has two commands, `build` and `local`. 

`local` will build the image, and install it into the local docker engine allow for it to be used on the local machine.

`build` will build the image and push it to docker hub.

To use these tasks from the current folder, you can use:

```shell
make python-local
make duckdb-local

make python-build
make duckdb-build 
```


## Build problems?

The makefiles provided attempt cross platform builds so that we are able to build on arm64 machines to be executed on amd64 machines. Depending on your setup, this may occassionally show the following error.

```
ERROR: Multiple platforms feature is currently not supported for docker driver. 
Please switch to a different driver (eg. "docker buildx create --use")
```

following the instructions given when you run `docker buildx create --use` should get you building again.