import datetime
import json
from dataclasses import asdict, dataclass
from typing import Dict, Optional, Type

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
from flytekit.extend import DictTransformer
from flytekit.models import literals
from flytekit.models.literals import LiteralMap
from flytekit.models.task import TaskTemplate
from flytekit.models.types import LiteralType, StructuredDatasetType

from bacalhau_sdk.api import submit, results
from bacalhau_sdk.config import get_client_id

import logging

# TODO (enricorotundo) add more metadata, api port and host, bacalhau dir, etc.
@dataclass
class Metadata:
    job_id: str
    
    def to_json(self):
        return json.dumps(self.__dict__)


class BacalhauAgent(AgentBase):
    """
    This agent submits a job to the Bacalhau API.
    All calls are idempotent
    """

    def __init__(self):
        # self.job_spec = job_spec
        # self.api_version = api_version
        self._logger = logging.getLogger(__name__)
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
            raise ValueError("inputs cannot be None")

        self._logger.debug(f"create inputs.literals: {inputs.literals}")
        inputs_dict = {}
        inputs_dict["api_version"] = inputs.literals.get("api_version").hash
        if inputs.literals.get("client_id") is not None:
            inputs_dict["client_id"] = inputs.literals.get("client_id").hash
        else:
            inputs_dict["client_id"] = get_client_id()
        inputs_dict["spec"] = inputs.literals.get("spec").hash
        self._logger.debug(f"create inputs_dict: {inputs_dict}")

        res = submit(
            dict(
                APIVersion=inputs_dict["api_version"],
                ClientID=inputs_dict["client_id"],
                Spec=inputs_dict["spec"],
            )
        )
        if not res:
            pass
        self._logger.debug(f"create res: {res}")
        metadata = Metadata(job_id=str(res.job.metadata.id))
        return CreateTaskResponse(resource_meta=json.dumps(asdict(metadata)).encode("utf-8"))

    def get(
        self, context: grpc.ServicerContext, resource_meta: bytes
    ) -> GetTaskResponse:
        metadata = Metadata(**json.loads(resource_meta.decode("utf-8")))
        baclhau_response = results(job_id=metadata.job_id)
        if not baclhau_response:
            self._logger.error("error")
            state = PERMANENT_FAILURE
            return GetTaskResponse(resource=Resource(state=state))
        
        state = SUCCEEDED
        ctx = FlyteContextManager.current_context()
        res = literals.LiteralMap(
            {
                "results": TypeEngine.to_literal(
                    ctx,
                    baclhau_response.job_id,
                    str,
                    literals.Literal.hash,
                )
            }
        ).to_flyte_idl()
        return GetTaskResponse(resource=Resource(state=state, outputs=res))


    def delete(
        self, context: grpc.ServicerContext, resource_meta: bytes
    ) -> DeleteTaskResponse:
        """ https://github.com/bacalhau-project/bacalhau/issues/2688 """
        return
    


AgentRegistry.register(BacalhauAgent())
