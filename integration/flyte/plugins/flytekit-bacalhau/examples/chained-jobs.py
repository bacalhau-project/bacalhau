"""
The first Bacalhau task of this workflow prints out "Flyte is awesome!" to stdout,
The second Bacalhau task is configured to take the output of the first Bacalhau task as input
and simply print that out.


https://docs.flyte.org/projects/cookbook/en/latest/auto_examples/control_flow/dynamics.html
"""

from flytekit import workflow, task, dynamic, kwtypes

from flytekitplugins.bacalhau import BacalhauTask


bacalhau_task = BacalhauTask(
    name="hello_world",
    inputs=kwtypes(
        spec=dict,
        api_version=str,
    ),
)


bacalhau_task_2 = BacalhauTask(
    name="second_hello_world",
    inputs=kwtypes(
        spec=dict,
        api_version=str,
    ),
)


@task
def second_task(bac_task: str) -> str:
    bac_task_2 = bacalhau_task_2(
        api_version="V1beta1",
        spec=dict(
            engine="Docker",
            verifier="Noop",
            PublisherSpec={"type": "IPFS"},
            docker={
                "image": "ubuntu",
                "entrypoint": ["cat", "/myinputs/stdout"],
            },
            language={"job_context": None},
            wasm=None,
            resources=None,
            timeout=1800,
            outputs=[
                {
                    "storage_source": "IPFS",
                    "name": "outputs",
                    "path": "/outputs",
                }
            ],
            inputs=[
                {
                    "cid": bac_task,
                    "name": "myinputs",
                    "path": "/myinputs",
                    "storageSource": "IPFS",
                }
            ],
            deal={"concurrency": 1},
        ),
    )

    return bac_task_2


@dynamic
def chained_job() -> str:
    bac_task_1 = bacalhau_task(
        api_version="V1beta1",
        spec=dict(
            engine="Docker",
            verifier="Noop",
            PublisherSpec={"type": "IPFS"},
            docker={
                "image": "ubuntu",
                "entrypoint": ["echo", "Flyte is awesome!"],
            },
            language={"job_context": None},
            wasm=None,
            resources=None,
            timeout=1800,
            outputs=[
                {
                    "storage_source": "IPFS",
                    "name": "outputs",
                    "path": "/outputs",
                }
            ],
            deal={"concurrency": 1},
        ),
    )

    return second_task(bac_task=bac_task_1)


@workflow
def wf() -> str:
    res = chained_job()
    return res


if __name__ == "__main__":
    wf()
