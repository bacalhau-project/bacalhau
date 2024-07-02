"""Example of how to list jobs."""

from bacalhau_sdk.job_store import JobStore

job_store = JobStore()
print(job_store.list())
