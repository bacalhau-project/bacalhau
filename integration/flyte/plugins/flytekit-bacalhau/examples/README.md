# flytekit-bacalhau examples

Prerequisite: build the provided Docker container locally with `make docker-build`.

## Hello World

Print a sample string to stdout. 

```shell
$ docker run --rm -v $(pwd):/examples \
        -w /examples \
		-a stdout -a stderr \
		winderresearch/flytekit-bacalhau-examples:latest \
		pyflyte run ./hello-world.py wf

QmcQnaoVtTaSBFukXD9yF8xfNYgQ8Yrf6UoxakpBBXZpd1
```

https://ipfs.io/ipfs/QmcQnaoVtTaSBFukXD9yF8xfNYgQ8Yrf6UoxakpBBXZpd1/

## Chain jobs together

Pipe task's A output into a downstream task B's input and have B process that.

```shell
$ docker run --rm -v $(pwd):/examples \
        -w /examples \
        -a stdout -a stderr \
		winderresearch/flytekit-bacalhau-examples:latest \
		pyflyte run ./chained-jobs.py wf

QmceCcBFqstn37YpJe4VMazYTEJ8moctdDXxqcDU9eFeMM
```

https://ipfs.io/ipfs/QmceCcBFqstn37YpJe4VMazYTEJ8moctdDXxqcDU9eFeMM/

---

## Troubleshoot

Issues with the jobs above? Try making Pyflyte verbose with `pyflyte --verbose run`.
