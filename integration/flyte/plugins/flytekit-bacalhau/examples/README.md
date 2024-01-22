# flytekit-bacalhau examples

## Prerequisite

If you have pip-installed `flytekitplugins-bacalhau` you should be able to run the examples directly with `pyflyte`.
However, for better reproducibility we provide a containerized environment you'd need to build first by executing `make docker-build`.

## 🌍 Hello World

Print a sample string to stdout. 
See the [full source code here](./hello-world.py).

```shell
$ docker run --rm -v $(pwd):/examples \
	-w /examples \
	-a stdout -a stderr \
	winderresearch/flytekit-bacalhau-examples:latest \
	pyflyte run ./hello-world.py wf

executing BacalhauTask with name: hello_world
job_id: c636e309-5d0c-4919-ad60-64ae82819bac resulted in cid: QmcQnaoVtTaSBFukXD9yF8xfNYgQ8Yrf6UoxakpBBXZpd1
```

https://ipfs.io/ipfs/QmcQnaoVtTaSBFukXD9yF8xfNYgQ8Yrf6UoxakpBBXZpd1/

## 🧑‍🤝‍🧑 Chain jobs together

Pipe task A's output into a downstream task B's input and have B process that.
See the [full source code here](./chained-jobs.py).

```shell
$ docker run --rm -v $(pwd):/examples \
	-w /examples \
	-a stdout -a stderr \
	winderresearch/flytekit-bacalhau-examples:latest \
	pyflyte run ./chained-jobs.py wf

executing BacalhauTask with name: upstream_task
executing BacalhauTask with name: downstream_task
job_id: e6175184-1263-4f8e-a7c3-e47e7c72a0eb resulted in cid: QmcQnaoVtTaSBFukXD9yF8xfNYgQ8Yrf6UoxakpBBXZpd1
job_id: 7a58c39c-1652-4c00-bafc-e0984bf6d87b resulted in cid: QmcQnaoVtTaSBFukXD9yF8xfNYgQ8Yrf6UoxakpBBXZpd1
```

https://ipfs.io/ipfs/QmceCcBFqstn37YpJe4VMazYTEJ8moctdDXxqcDU9eFeMM/

---

## Troubleshoot

Issues with the jobs above? Try making Pyflyte verbose with `pyflyte --verbose run`.
