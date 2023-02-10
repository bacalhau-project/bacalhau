"""
Airflow hook to interact with the Bacalhau service.
Hooks should be completely state-less.
"""
from bacalhau_sdk.api import submit, results, events
from bacalhau_sdk.config import get_client_id

from airflow.hooks.base import BaseHook


class BacalhauHook(BaseHook):
    """Hook to interact with the Bacalhau service."""

    def __init__(self, **kwargs) -> None:
        super().__init__(**kwargs)
        self.client_id = get_client_id()

    def submit_job(self, api_version: str, job_spec: dict) -> dict:
        """Submit a job to the Bacalhau service.
        Returns the job ID.
        """
        response = submit(dict(
            apiversion=api_version,
            clientid=self.client_id,
            spec=job_spec,
        ))
        # TODO check if response is not empty
        # TODO return dict
        return str(response.job.metadata.id)

    def get_results(self, job_id: str) -> list:
        """Get the results of a job."""
        response = results(job_id)
        # TODO check if response is not empty
        return response.to_dict()['results']
        
    def get_events(self, job_id: str) -> dict:
        """Get the events of a job."""
        response = events(job_id)
        # TODO check if response is not empty
        return response.to_dict()