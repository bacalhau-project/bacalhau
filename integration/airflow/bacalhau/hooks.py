from bacalhau_sdk.api import submit
from bacalhau_sdk.config import get_client_id

from airflow.hooks.base import BaseHook


class BacalhauHook(BaseHook):
    """Hook to interact with the Bacalhau service."""

    def __init__(self, **kwargs) -> None:
        super().__init__(**kwargs)
        self.client_id = get_client_id()

    def submit_job(self, api_version: str, job_spec: dict) -> dict:
        """Submit a job to the Bacalhau service."""
        response = submit(dict(
            apiversion=api_version,
            clientid=self.client_id,
            spec=job_spec,
        ))
        return str(response)
