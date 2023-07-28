"""
pyflyte run scripts/wf.py bacalhau_task
"""

import bacalhau_sdk
print(bacalhau_sdk)
print(bacalhau_sdk.config.get_client_id())

from flytekit import workflow

from flytekitplugins.bacalhau import BacalhauTask


bacalhau_task = BacalhauTask(
    name="hello_world",
    api_version="V1beta1",
    job_spec=dict(
        engine="Docker",
        verifier="Noop",
        PublisherSpec={"type": "IPFS"},
        docker={
            "image": "ubuntu",
            "entrypoint": ["echo", "Hello World!"],
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
        deal={"concurrency": 1, "confidence": 0, "min_bids": 0},
        do_not_track=True,
    )
)

@workflow
def wf() -> str:
    return bacalhau_task(myinput="myinput")


if __name__ == "__main__":
    print(wf())

