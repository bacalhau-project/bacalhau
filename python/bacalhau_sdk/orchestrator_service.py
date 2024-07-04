from bacalhau_apiclient.configuration import Configuration
from bacalhau_apiclient.api import orchestrator_api
from bacalhau_apiclient.models.api_put_job_request import (
    ApiPutJobRequest as PutJobRequest,
)
from bacalhau_apiclient.models.api_put_job_response import (
    ApiPutJobResponse as PutJobResponse,
)
from bacalhau_apiclient.models.api_stop_job_response import (
    ApiStopJobResponse as StopJobResponse,
)
from bacalhau_apiclient.models.api_list_job_results_response import (
    ApiListJobResultsResponse as ListJobResultsResponse,
)
from bacalhau_apiclient.models.api_list_job_executions_response import (
    ApiListJobExecutionsResponse as ListJobExecutionsResponse,
)
from bacalhau_apiclient.models.api_list_jobs_response import (
    ApiListJobsResponse as ListJobsResponse,
)
from bacalhau_apiclient.models.api_list_job_history_response import (
    ApiListJobHistoryResponse as ListJobHistoryResponse,
)
from bacalhau_apiclient.rest import ApiException
from bacalhau_apiclient.models.api_get_job_response import (
    ApiGetJobResponse as GetJobResponse,
)


class OrchestratorService:
    def __init__(self, config: Configuration):
        self.api_client = orchestrator_api.ApiClient(config)
        self.endpoint = orchestrator_api.OrchestratorApi(self.api_client)

    def put_job(self, request: PutJobRequest) -> PutJobResponse:
        try:
            return self.endpoint.orchestratorput_job(request)
        except ApiException as e:
            print("Exception while calling OrchestratoApi->putJob %s\n" % e)

    def stop_job(self, id: str, reason: str) -> StopJobResponse:
        try:
            return self.endpoint.orchestratorstop_job(id=id, reason=reason)
        except ApiException as e:
            print("Exception while calling OrchestratirApi->stopJob %s\n" % e)

    def job_results(
        self,
        id: str,
    ) -> ListJobResultsResponse:
        try:
            return self.endpoint.orchestratorjob_results(
                id=id,
            )
        except ApiException as e:
            print("Exception while calling OrchestratirApi->jobResults %s\n" % e)

    def get_job(
        self,
        id: str,
        include: str = "",
        limit: int = 10,
    ) -> GetJobResponse:
        try:
            return self.endpoint.orchestratorget_job(
                id=id, include=include, limit=limit
            )
        except ApiException as e:
            print("Exception while calling OrchestratirApi->getJob %s\n" % e)

    def job_executions(
        self,
        id: str,
        namespace: str = "",
        limit: int = 5,
        next_token: str = "",
        reverse: bool = False,
        order_by: str = "",
    ) -> ListJobExecutionsResponse:
        try:
            return self.endpoint.orchestratorjob_executions(
                id=id,
                namespace=namespace,
                limit=limit,
                next_token=next_token,
                reverse=reverse,
                order_by=order_by,
            )
        except ApiException as e:
            print("Exception while calling OrchestratirApi->jobExecutions %s\n" % e)

    def job_history(
        self,
        id: str,
        event_type: str = "execution",
        node_id: str = "",
        execution_id: str = "",
    ) -> ListJobHistoryResponse:
        try:
            return self.endpoint.orchestratorjob_history(
                id=id,
                event_type=event_type,
                node_id=node_id,
                execution_id=execution_id,
            )
        except ApiException as e:
            print("Exception while calling OrchestratirApi->jobHistory %s\n" % e)

    def list_jobs(
        self,
        limit: int = 5,
        next_token: str = "",
        order_by: str = "created_at",
        reverse: bool = False,
    ) -> ListJobsResponse:
        try:
            api_response = self.endpoint.orchestratorlist_jobs(
                limit=limit,
                order_by=order_by,
                reverse=reverse,
                next_token=next_token,
            )
        except ApiException as e:
            print("Exception while calling OrchestratirApi->listJobs %s\n" % e)

        return api_response
