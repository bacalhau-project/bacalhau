"""
The first Bacalhau task of this workflow prints out "Flyte is awesome!" to stdout,
The second Bacalhau task is configured to take the output of the first Bacalhau task as input
and simply print that out.


https://docs.flyte.org/projects/cookbook/en/latest/auto_examples/control_flow/dynamics.html
"""

from flytekit import workflow, task, dynamic, kwtypes

from flytekitplugins.bacalhau import BacalhauTask


bacalhau_task_1 = BacalhauTask(
    name="upstream_task",
    inputs=kwtypes(
        spec=dict,
        api_version=str,
    ),
)


bacalhau_task_2 = BacalhauTask(
    name="downstream_task",
    inputs=kwtypes(
        spec=dict,
        api_version=str,
    ),
)


@task
def resolve_task(bac_task: str) -> str:
    task_2 = bacalhau_task_2(
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

    return task_2


@dynamic
def chain_jobs() -> str:
    task_1 = bacalhau_task_1(
        api_version="V1beta1",
        spec=dict(
            engine="Docker",
            verifier="Noop",
            PublisherSpec={"type": "IPFS"},
            docker={
                "image": "ubuntu",
                "entrypoint": ["echo", "Flyte is awesome and with Bacalhau it's even better!"],
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

    return resolve_task(bac_task=task_1)


@workflow
def wf():
    chain_jobs()

if __name__ == "__main__":
    wf()
