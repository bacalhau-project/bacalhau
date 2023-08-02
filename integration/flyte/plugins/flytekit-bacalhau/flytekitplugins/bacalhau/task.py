from flytekit.configuration import SerializationSettings
from flytekit.extend import Interface, PythonTask, context_manager
from flytekit.extend.backend.base_agent import AsyncAgentExecutorMixin
from flytekit import kwtypes


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
        task_config=None,
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

    
