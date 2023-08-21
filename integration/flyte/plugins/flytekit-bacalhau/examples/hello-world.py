"""
The Bacalhau task of this workflow prints out "Flyte is awesome!" to stdout.
As simple as that.
"""

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

# https://docs.flyte.org/projects/cookbook/en/latest/auto_examples/basics/basic_workflow.html#how-does-a-flyte-workflow-work
@task
def print_cid(bac_task: str) -> str:
    print(f"Your Bacalhau's output CID: {bac_task}")
    return bac_task

@workflow
def wf() -> str:
    bac_task = bacalhau_task(
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
    
    #return 
    #print_cid(bac_task=bac_task)
    return bac_task


if __name__ == "__main__":
    wf()
