"""
Airflow hook to interact with the Bacalhau service.
"""
from bacalhau_sdk.api import events, results, submit
from bacalhau_sdk.config import get_client_id

from airflow.hooks.base import BaseHook


class BacalhauHook(BaseHook):
    """Hook to interact with the Bacalhau service."""

    def __init__(self, **kwargs) -> None:
        """
        Initialize the hook.

        Args:
            kwargs: Additional keyword arguments.
        """
        super().__init__(**kwargs)
        self.client_id = get_client_id()

    def submit_job(self, api_version: str, job_spec: dict) -> str:
        """Submit a job to the Bacalhau service.

        Args:
            api_version (str): The API version to use. Example: "V1beta1".
            job_spec (dict): A dictionary with the job specification. See example dags for more details.

        Returns:
            str: The job ID. Example: "3b39baee-5714-4f17-aa71-1f5824665ad6".
        """

        response = submit(
            dict(
                apiversion=api_version,
                clientid=self.client_id,
                spec=job_spec,
            )
        )
        # TODO check if response is not empty
        return str(response.job.metadata.id)

    def get_results(self, job_id: str) -> list:
        """Get the data generated from a job. The data becomes available only after the job is finished.

        Args:
            job_id (str): The job ID to get the results from. Example: "3b39baee-5714-4f17-aa71-1f5824665ad6".

        Returns:
            list: A list of dictionaries with the results, one entry per node & shard pair. A nested field contains a CID pointer to the result data.
        """
        response = results(job_id)
        # TODO check if response is not empty
        return response.to_dict()["results"]

    def get_events(self, job_id: str) -> dict:
        """Get the events of a job. This is useful to check its status.

        Args:
            job_id (str): The job ID to get the events from. Example: "3b39baee-5714-4f17-aa71-1f5824665ad6".

        Returns:
            dict: List of dictionaries with the events
        """
        response = events(job_id)
        # TODO check if response is not empty
        return response.to_dict()
