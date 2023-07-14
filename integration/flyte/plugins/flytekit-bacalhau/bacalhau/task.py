# import pprint
import os
from dataclasses import dataclass
from typing import Any, Callable
from typing import Any

from flytekit import PythonFunctionTask
from flytekit.extend import PythonTask
from flytekit.extend import Interface, PythonTask, context_manager

from bacalhau_sdk.api import submit
from bacalhau_sdk.config import get_client_id


@dataclass
class BacalhauConfig(object):
    """
    BacalhauConfig should be used to configure a Bacalhau Task.
    e.g. point to a Bacalhau API endpoint, or set a timeout.
    """
    pass


class BacalhauTask(PythonTask):
    """
    BacalhauTask is a Flyte task that submits a job to the Bacalhau API.

    https://docs.flyte.org/projects/flytekit/en/latest/generated/flytekit.core.python_function_task.PythonFunctionTask.html#flytekit-core-python-function-task-pythonfunctiontask
    """
    
    _TASK_TYPE = "bacalhau"
    job_spec: dict
    myinput: str = "myinput"
    myoutput: str = "myoutput"
    api_version: str
    client_id: str

    def __init__(
        self, 
        name: str,
        job_spec: dict, # TODO: make this a BacalhauConfig
        api_version: str = "V1beta1",
        client_id: str = get_client_id(),
        # task_config: BacalhauConfig,
        **kwargs,
    ):

        self.job_spec = job_spec
        self.api_version = api_version
        self.client_id = client_id

        super(BacalhauTask, self).__init__(
            task_type=self._TASK_TYPE,
            name=name,
            task_config=None,
            interface=Interface(
                inputs={self.myinput: str}, outputs={self.myoutput: str}
            ),
            **kwargs,
        )

    def execute(self, **kwargs) -> Any:
        # No need to check for existence, as that is guaranteed.
        ctx = context_manager.FlyteContext.current_context()
        user_context = ctx.user_space_params
        user_context.logging.info("Calling Bac API...")

        if "annotations" in self.job_spec:
            self.job_spec["annotations"] = self.job_spec["annotations"].append("flyte")
        else:
            self.job_spec["annotations"] = ["flyte"]

        data = {
            "Spec": self.job_spec,
            "APIVersion": self.api_version,
            "ClientID": self.client_id,
        }
        
        return submit(data).job.metadata.id

    # def get_config(self) -> BacalhauConfig:
    #     return self._task_config
