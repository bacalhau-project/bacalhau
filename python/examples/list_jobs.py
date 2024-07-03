"""Example of how to list jobs."""

from bacalhau_sdk.jobs import Jobs

jobs = Jobs()
print(jobs.list())
