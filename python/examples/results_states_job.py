"""Example of how to get the results and states of a job."""

from bacalhau_sdk.api import results, states

print("Results:")
print(results(job_id="655ef2a7-604d-4799-8eef-9b848914d101"))

print("States:")
print(states(job_id="655ef2a7-604d-4799-8eef-9b848914d101"))
