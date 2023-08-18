from flytekit import workflow

from flytekitplugins.bacalhau import BacalhauTask
from flytekit import kwtypes
from flytekit.core.promise import get_primitive_val

bacalhau_task = BacalhauTask(
        name="hello_world",
        inputs=kwtypes(
                spec=dict,
                api_version=str,
        )
    )


@workflow
def wf():
    a = bacalhau_task(
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
            do_not_track=True,
        ),
    )
    print(type(a))
    print(a)

if __name__ == "__main__":
    wf()
