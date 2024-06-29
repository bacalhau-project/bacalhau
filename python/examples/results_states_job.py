"""Example of how to get the results and states of a job."""

from bacalhau_sdk.job_store import JobStore

job_store = JobStore()
print("Results:")
print(job_store.results(job_id="655ef2a7-604d-4799-8eef-9b848914d101"))

print("Executions")
print(job_store.executions(job_id="655ef2a7-604d-4799-8eef-9b848914d101"))
