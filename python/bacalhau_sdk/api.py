"""Submit a job to the server."""

from bacalhau_apiclient.models.job import Job
from bacalhau_apiclient.models.api_put_job_request import (
    ApiPutJobRequest as PutJobRequest,
)
from bacalhau_sdk.config import (
    init_config,
)
from bacalhau_sdk.orchestrator_service import OrchestratorService

conf = init_config()
orchestrator_service = OrchestratorService(conf)


def submit(job: Job):
    """Submit a job to the server."""
    request = PutJobRequest(job=job)
    return orchestrator_service.put_job(request=request)


def cancel(job_id: str):
    """Cancels a job on the server."""
    return orchestrator_service.stop_job(job_id, "UserCancelled")


def list():
    """List all jobs."""
    return orchestrator_service.list_jobs()


def results(job_id: str):
    """Return Job Results"""
    return orchestrator_service.job_results(id=job_id)


def states(job_id: str):
    """Return Job States"""
    return orchestrator_service.job_executions(id=job_id)


def events(job_id: str):
    """Returns Job Related Events"""
    return orchestrator_service.job_history(id=job_id)
