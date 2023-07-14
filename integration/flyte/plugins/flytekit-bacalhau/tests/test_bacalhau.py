import pytest
import os
from flytekitplugins.bacalhau import BacalhauTask
from flytekitplugins.bacalhau.task import BacalhauConfig
from flytekit import workflow

def test_config():
    config = BacalhauConfig()
    assert config.BacalhauApiHost is None
    assert os.environ.get("BACALHAU_API_HOST") is None

    foo_host = "http://foo"
    config = BacalhauConfig(
        BacalhauApiHost=foo_host
    )
    assert config.BacalhauApiHost == foo_host
    assert os.environ["BACALHAU_API_HOST"] == foo_host

def test_local_exec():
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
        return bacalhau_task(myinput="hello")