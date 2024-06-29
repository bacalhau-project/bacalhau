import pprint

from bacalhau_apiclient.models.job import Job
from bacalhau_apiclient.models.task import Task
from bacalhau_apiclient.models.all_of_execution_published_result import SpecConfig
from bacalhau_apiclient.models.api_put_job_request import (
    ApiPutJobRequest as PutJobRequest,
)
from bacalhau_sdk.job_store import JobStore

task = Task(
    name="My Main task",
    engine=SpecConfig(
        type="docker",
        params=dict(
            Image="ubuntu:latest",
            Entrypoint=["/bin/bash"],
            Parameters=["-c", "echo Hello World"],
        ),
    ),
    publisher=SpecConfig(),
)

job = Job(name="A Simple Docker Job", type="batch", count=1, tasks=[task])
put_job_request = PutJobRequest(job=job)
job_store = JobStore()

pprint.pprint("Submitted Job Response")
put_job_response = job_store.put(put_job_request)
pprint.pprint(put_job_response)


pprint.pprint("Get Job Response With Executions")
pprint.pprint(job_store.get(put_job_response.job_id, include="executions"))
