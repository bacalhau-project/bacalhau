"""Example of how to get the results and states of a job."""

from bacalhau_sdk.jobs import Jobs

jobs = Jobs()
print("Results:")
print(jobs.results(job_id="655ef2a7-604d-4799-8eef-9b848914d101"))

print("Executions")
print(jobs.executions(job_id="655ef2a7-604d-4799-8eef-9b848914d101"))
