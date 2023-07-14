import pytest
import os
from flytekitplugins.bacalhau import BacalhauTask
from flytekitplugins.bacalhau.task import BacalhauConfig
from flytekit import workflow

from bacalhau_sdk.api import submit
from bacalhau_sdk.config import get_client_id
from bacalhau_apiclient.models.spec import Spec

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
    # TODO - make sure this test has no side effects
    bacalhau_task = BacalhauTask(
        name="test",
        api_version="V1beta1",
        job_spec=dict(
            engine="Docker",
            verifier="Noop",
            PublisherSpec={"type": "IPFS"},
            docker={
                "image": "ubuntu",
                "entrypoint": ["echo", "This is a test"],
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
        )
    )

    # TODO check interface
    # bacalhau_task

    # @workflow
    # def wf() -> str:
    #     return bacalhau_task(myinput="hello")
    
    job_id = bacalhau_task()
    assert job_id is not None
    assert isinstance(job_id, str)
    assert len(job_id) == 36, "job_id should be a uuid"