# flytekit-bacalhau examples

```shell
# build docker container
$ make docker-build

$ docker run --rm -v $(pwd):/examples \
        -w /examples \
		winderresearch/flytekit-bacalhau-examples:latest \
		pyflyte run ./hello-world.py wf

$ docker run --rm -v $(pwd):/examples \
        -w /examples \
        -a stdout -a stderr \
		winderresearch/flytekit-bacalhau-examples:latest \
		pyflyte run --verbose ./chained-jobs.py wf
```