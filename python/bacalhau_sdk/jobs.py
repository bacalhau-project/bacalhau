from bacalhau_sdk.orchestrator_service import OrchestratorService
from bacalhau_apiclient.models.api_put_job_response import (
    ApiPutJobResponse as PutJobResponse,
)
from bacalhau_apiclient.models.api_put_job_request import (
    ApiPutJobRequest as PutJobRequest,
)
from bacalhau_apiclient.models.api_stop_job_response import (
    ApiStopJobResponse as StopJobResponse,
)
from bacalhau_apiclient.models.api_list_job_executions_response import (
    ApiListJobExecutionsResponse as ListJobExecutionsResponse,
)
from bacalhau_apiclient.models.api_list_job_results_response import (
    ApiListJobResultsResponse as ListJobResultsResponse,
)

from bacalhau_apiclient.models.api_list_job_history_response import (
    ApiListJobHistoryResponse as ListJobHistoryResponse,
)
from bacalhau_apiclient.models.api_list_jobs_response import (
    ApiListJobsResponse as ListJobsResponse,
)
from bacalhau_apiclient.models.api_get_job_response import (
    ApiGetJobResponse as GetJobResponse,
)

from bacalhau_sdk.config import init_config


class Jobs:
    def __init__(self):
        self.orchestrator_service = OrchestratorService(config=init_config())

    def put(self, request: PutJobRequest) -> PutJobResponse:
        """Puts Job On Bacalhau Network

        Args:
            request (PutJobRequest): A request to put a job to bacalhau network. It encapsulates the job model.

        Returns:
            PutJobResponse: Once job is successful put on bacalhau netowork, this returns the job details.
        """
        return self.orchestrator_service.put_job(request)

    def stop(self, job_id: str, reason: str = None) -> StopJobResponse:
        return self.orchestrator_service.stop_job(id=job_id, reasone=reason)

    def executions(
        self,
        job_id: str,
        namespace: str = "",
        next_token: str = "",
        limit: int = 5,
        reverse: bool = False,
        order_by: str = "",
    ) -> ListJobExecutionsResponse:
        """Gets Job Executions

        Args:
            job_id (str): The id of the job for executions are being fetched.
            namespace (str, optional): Namespace to which the job belongs to Defaults to "".
            next_token (str, optional): Next Page Token. Defaults to "".
            limit (int, optional): The number of executions to fetch at most. Defaults to 5.
            reverse (bool, optional): Should the order_by be reveresed Defaults to False.
            order_by (str, optional): Order the executions by certain property Defaults to "".

        Returns:
            ListJobExecutionsResponse: A list of job's execution(s)
        """

        return self.orchestrator_service.job_executions(
            id=job_id,
            namespace=namespace,
            limit=limit,
            next_token=next_token,
            reverse=reverse,
            order_by=order_by,
        )

    def results(
        self,
        job_id: str,
    ) -> ListJobResultsResponse:
        """_

        Args:
            id (str): The job id for which results are being fetched.

        Returns:
            ListJobResultsResponse: A list of job's result(s)
        """
        return self.orchestrator_service.job_results(id=id)

    def get(self, job_id: str, include: str = "", limit: int = 10) -> GetJobResponse:
        """Get Details of a Job

        Args:
            job_id (str): The job id for which the details are being fetched.
            include (str, optional): Whether to include executions or history details Defaults to "".
            limit (int, optional): The number of executions or history to limit by. Defaults to 10.

        Returns:
            GetJobResponse: The details of the job.
        """
        return self.orchestrator_service.get_job(
            id=job_id, include=include, limit=limit
        )

    def history(
        self,
        job_id: str,
        event_type: str = "execution",
        node_id: str = "",
        execution_id: str = "",
    ) -> ListJobHistoryResponse:
        """Get History of a Job

        Args:
            job_id (str): The job id for which the history is being fetched.
            event_type (str, optional): Type of event to fetch history for. Defaults to "execution".
            node_id (str, optional): Fetch history for the job_id being executed on particular node_id. Defaults to "".
            execution_id (str, optional): Fetch history for particular execution of job. Defaults to "".

        Returns:
            ListJobHistoryResponse: _description_
        """
        return self.orchestrator_service.job_history(
            id=job_id, event_type=event_type, node_id=node_id, execution_id=execution_id
        )

    def list(
        self,
        limit: int = 5,
        next_token: str = "",
        order_by: str = "created_at",
        reverse: bool = False,
    ) -> ListJobsResponse:
        """List Jobs

        Args:
            limit (int, optional): Number of jobs to list Defaults to 5.
            next_token (str, optional): The next page token. Defaults to "".
            order_by (str, optional): Order jobs by particulare property . Defaults to "created_at".
            reverse (bool, optional): Should the ordering be reversed. Defaults to False.

        Returns:
            ListJobsResponse: _description_
        """
        return self.orchestrator_service.list_jobs(
            limit=limit, next_token=next_token, order_by=order_by, reverse=reverse
        )
