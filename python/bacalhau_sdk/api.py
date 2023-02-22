"""Submit a job to the server."""

import json

from bacalhau_apiclient.api import job_api
from bacalhau_apiclient.models.list_request import ListRequest
from bacalhau_apiclient.models.state_request import StateRequest
from bacalhau_apiclient.models.events_request import EventsRequest
from bacalhau_apiclient.models.submit_request import SubmitRequest
from bacalhau_apiclient.rest import ApiException

from bacalhau_sdk.config import get_client_id, get_client_public_key, init_config, sign_for_client

conf = init_config()
client = job_api.ApiClient(conf)
api_instance = job_api.JobApi(client)


def submit(data: dict):
    """Submit a job to the server.

    Input `data` object is sanittized and signed before being sent to the server.
    """
    sanitized_data = client.sanitize_for_serialization(data)
    json_data = json.dumps(sanitized_data, indent=None, separators=(", ", ": "))
    json_bytes = json_data.encode("utf-8")
    signature = sign_for_client(json_bytes)
    client_public_key = get_client_public_key()
    submit_req = SubmitRequest(
        client_public_key=client_public_key,
        job_create_payload=sanitized_data,
        signature=signature,
    )
    return api_instance.submit(submit_req)


def list():
    """List all jobs."""
    try:
        # Simply lists jobs.
        list_request = ListRequest(
            client_id=get_client_id(),
            sort_reverse=False,
            sort_by="created_at",
            return_all=False,
            max_jobs=5,
            exclude_tags=[],
            include_tags=[],
        )
        api_response = api_instance.list(list_request)
    except ApiException as e:
        print("Exception when calling JobApi->list: %s\n" % e)
    return api_response


def results(job_id: str):
    """Get results."""
    try:
        # Returns the results of the job-id specified in the body payload.
        state_request = StateRequest(
            client_id=get_client_id(),
            job_id=job_id,
        )
        api_response = api_instance.results(state_request)
    except ApiException as e:
        print("Exception when calling JobApi->results: %s\n" % e)
    return api_response


def states(job_id: str):
    """Get states."""
    try:
        # Returns the state of the job-id specified in the body payload.
        state_request = StateRequest(
            client_id=get_client_id(),
            job_id=job_id,
        )
        api_response = api_instance.states(state_request)
    except ApiException as e:
        print("Exception when calling JobApi->states: %s\n" % e)
    return api_response


def events(job_id: str):
    """Get events."""
    # TODO - add tests
    try:
        # Returns the events of the job-id specified in the body payload.
        state_request = EventsRequest(
            client_id=get_client_id(),
            job_id=job_id,
        )
        api_response = api_instance.events(state_request)
    except ApiException as e:
        print("Exception when calling JobApi->events: %s\n" % e)
    return api_response
