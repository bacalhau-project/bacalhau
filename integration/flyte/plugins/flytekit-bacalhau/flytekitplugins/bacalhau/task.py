# import pprint
import os
from dataclasses import dataclass
from typing import Any, Dict, Optional, Type

from google.protobuf import json_format
from google.protobuf import struct_pb2

from flytekit.configuration import SerializationSettings
from flytekit.extend import Interface, PythonTask, context_manager
from flytekit.extend.backend.base_agent import AsyncAgentExecutorMixin
from flytekit import kwtypes

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

    def __init__(
        self,
        name: str,
        task_config=Optional[BacalhauConfig],
        **kwargs,
    ):
        """
        To be used to run a Bacalhau job.

        :param name: Name of this task
        :param task_config: BacalhauConfig object
        :param kwargs: All other args required by Parent type - PythonTask
        """
        interface = Interface(
            inputs=kwtypes(
                spec=dict,
                api_version=str,
            ),
            outputs=kwtypes(results=str)
        )
        super(BacalhauTask, self).__init__(
            task_type=self._TASK_TYPE,
            name=name,
            task_config=task_config,
            interface=interface,
            # environment put ENV VAR into this param?
            **kwargs,
        )

    def get_custom(self, settings: SerializationSettings) -> Dict[str, Any]:
        config = {
            "BacalhauApiHost": self.task_config.BacalhauApiHost,
            "BacalhauApiPort": self.task_config.BacalhauApiPort,
            "BacalhauDir": self.task_config.BacalhauDir,
        }
        s = struct_pb2.Struct()
        s.update(config)
        return json_format.MessageToDict(s)
    
