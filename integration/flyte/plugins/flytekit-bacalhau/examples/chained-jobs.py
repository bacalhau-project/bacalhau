from flytekit import workflow, task

from flytekitplugins.bacalhau import BacalhauTask
from flytekit import kwtypes


bacalhau_task = BacalhauTask(
        name="hello_world",
        inputs=kwtypes(
                spec=dict,
                api_version=str,
        )
    )

bacalhau_task_2 = BacalhauTask(
        name="second_hello_world",
        inputs=kwtypes(
                spec=dict,
                api_version=str,
        )
    )

# https://docs.flyte.org/projects/cookbook/en/latest/auto_examples/basics/basic_workflow.html#how-does-a-flyte-workflow-work
@task
def print_cid(bac_task: str) -> str:
    print(f"Your Bacalhau's output CID: {bac_task}")
    return bac_task

@workflow
def wf():
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
            outputs=[{
                    "storage_source": "IPFS",
                    "name": "outputs",
                    "path": "/outputs",
            }],
            deal={"concurrency": 1},
        ),
    )

    bac_task_2 = bacalhau_task_2(
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
            outputs=[{
                    "storage_source": "IPFS",
                    "name": "outputs",
                    "path": "/outputs",
            }],
            inputs=[{
                    "storage_source": "IPFS",
                    "cid": bac_task_1,
                    "name": "myinputs",
                    "path": "/myinputs",
            }],
            deal={"concurrency": 1},
        ),
    )
    
    print_cid(bac_task=bac_task_2)


if __name__ == "__main__":
    wf()
