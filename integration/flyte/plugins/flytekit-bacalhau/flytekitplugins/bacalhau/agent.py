import datetime
import json
from dataclasses import asdict, dataclass
from typing import Dict, Optional

import grpc
from flyteidl.admin.agent_pb2 import (
    PERMANENT_FAILURE,
    SUCCEEDED,
    CreateTaskResponse,
    DeleteTaskResponse,
    GetTaskResponse,
    Resource,
)

from flytekit import FlyteContextManager, StructuredDataset, logger
from flytekit.core.type_engine import TypeEngine
from flytekit.extend.backend.base_agent import (
    AgentBase,
    AgentRegistry,
    convert_to_flyte_state,
)
from flytekit.models import literals
from flytekit.models.literals import LiteralMap
from flytekit.models.task import TaskTemplate
from flytekit.models.types import LiteralType, StructuredDatasetType

from bacalhau_sdk.api import submit, results
from bacalhau_sdk.config import get_client_id


@dataclass
class Metadata:
    job_id: str
    # TODO (enricorotundo) add more metadata, api port and host, bacalhau dir, etc.


class BacalhauAgent(AgentBase):
    """
    This agent submits a job to the Bacalhau API.
    All calls are idempotent
    """

    def __init__(self):
        # self.job_spec = job_spec
        # self.api_version = api_version
        super().__init__(task_type="bacalhau_task")

    def create(
        self,
        context: grpc.ServicerContext,
        output_prefix: str,
        task_template: TaskTemplate,
        inputs: Optional[LiteralMap] = None,
    ) -> CreateTaskResponse:
        """_summary_

        Spec(
                    engine="Docker",
                    verifier="Noop",
                    publisher_spec={"type": "Estuary"},
                    docker={
                        "image": "ubuntu",
                        "entrypoint": ["echo", "Hello World!"],
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
                        },
                    ],
                    deal={"concurrency": 1},
                    do_not_track=False,
                )

        Args:
            context (grpc.ServicerContext): _description_
            output_prefix (str): _description_
            task_template (TaskTemplate): _description_
            inputs (Optional[LiteralMap], optional): _description_. Defaults to None.

        Returns:
            CreateTaskResponse: _description_
        """

        if not inputs:
            pass

        if inputs.get("client_id") is None:
            client_id = get_client_id()
        else:
            client_id = inputs["client_id"]

        res = submit(
            dict(
                APIVersion=inputs["api_version"],
                ClientID=client_id,
                Spec=inputs["spec"],
            )
        )
        if not res:
            pass
        metadata = Metadata(job_id=str(res.job.metadata.id))
        return CreateTaskResponse(resource_meta=json.dumps(metadata).encode("utf-8"))

    def get(
        self, context: grpc.ServicerContext, resource_meta: bytes
    ) -> GetTaskResponse:
        metadata = Metadata(**json.loads(resource_meta.decode("utf-8")))
        # res = requests.get(url, json={"job_id": metadata.job_id})
        res = results(job_id=metadata.job_id)
        return GetTaskResponse(resource=Resource(state=res.state))


    def delete(
        self, context: grpc.ServicerContext, resource_meta: bytes
    ) -> DeleteTaskResponse:
        print("is this implemented?")
        return
    


AgentRegistry.register(BacalhauAgent())
