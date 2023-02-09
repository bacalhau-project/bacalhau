from airflow.models import BaseOperator
from bacalhau.hooks import BacalhauHook
from airflow.compat.functools import cached_property


class BacalhauSubmitJobOperator(BaseOperator):
    """Submit a job to the Bacalhau service."""

    def __init__(self,
                 api_version: str,
                 job_spec: dict,
                 **kwargs) -> None:
        super().__init__(**kwargs)
        self.api_version = api_version
        self.job_spec = job_spec

    def execute(self, context) -> dict:
        job_create_response = self.hook.submit_job(
            api_version=self.api_version, job_spec=self.job_spec)
        return job_create_response

    @cached_property
    def hook(self) -> BacalhauHook:
        """Create and return an BacalhauHook (cached)."""
        return BacalhauHook()

    def get_hook(self) -> BacalhauHook:
        """Create and return an BacalhauHook (cached)."""
        return self.hook
