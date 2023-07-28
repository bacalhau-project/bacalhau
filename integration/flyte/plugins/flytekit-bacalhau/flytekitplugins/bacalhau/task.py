# import pprint
import os
from dataclasses import dataclass
from typing import Any, Dict, Optional, Type

from flytekit.extend import Interface, PythonTask, context_manager
from flytekit.extend.backend.base_agent import AsyncAgentExecutorMixin


@dataclass
class BacalhauConfig(object):
    """
    BacalhauConfig should be used to configure a Bacalhau Task.
    i.e., point to a Bacalhau API endpoint, port, or a Bacalhau directory.
    """
    BacalhauApiHost: Optional[str] = None
    BacalhauApiPort: Optional[str] = None
    BacalhauDir: Optional[str] = None

    def __post_init__(self):
        if self.BacalhauApiHost is not None:
            os.environ["BACALHAU_API_HOST"] = self.BacalhauApiHost
        if self.BacalhauApiPort is not None:
            os.environ["BACALHAU_API_PORT"] = self.BacalhauApiPort
        if self.BacalhauDir is not None:
            os.environ["BACALHAU_DIR"] = self.BacalhauDir

class BacalhauTask(AsyncAgentExecutorMixin, PythonTask):
    """
    This task submits a job to Bacalhau (https://github.com/bacalhau-project/bacalhau).
    """
    
    _TASK_TYPE = "bacalhau_task"

    job_spec: dict
    api_version: str
    client_id: str
    
    # TODO - make these configurable?
    myinput: str = "myinput"
    myoutput: str = "myoutput"

    def __init__(
        self,
        name: str,
        job_spec: dict,
        api_version: str = "V1beta1",
        **kwargs,
    ):

        self.job_spec = job_spec
        self.api_version = api_version

        super(BacalhauTask, self).__init__(
            task_type=self._TASK_TYPE,
            name=name,
            task_config=None,
            interface=Interface(
                inputs={self.myinput: str}, outputs={self.myoutput: str}
            ),
            # environment put ENV VAR into this param?
            **kwargs,
        )

    # def execute(self, **kwargs) -> Any:
    #     # No need to check for existence, as that is guaranteed.
    #     ctx = context_manager.FlyteContext.current_context()
    #     user_context = ctx.user_space_params
    #     user_context.logging.info("Calling Bacalhau API...")

    #     if "annotations" in self.job_spec:
    #         self.job_spec["annotations"] = self.job_spec["annotations"].append("flyte")
    #     else:
    #         self.job_spec["annotations"] = ["flyte"]

    #     data = {
    #         "Spec": self.job_spec,
    #         "APIVersion": self.api_version,
    #         "ClientID": self.client_id,
    #     }
        
    #     return submit(data).job.metadata.id

    # def get_config(self) -> BacalhauConfig:
    #     return self._task_config
