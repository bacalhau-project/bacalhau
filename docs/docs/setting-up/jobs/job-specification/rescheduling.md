---

sidebar_label: Rescheduling
---

# Rescheduling Policy Specification

The `ReschedulingPolicy` object provides a mechanism to control repeated attempts to schedule a job that cannot be completely scheduled at the first attempt.

## `ReschedulingPolicy` Parameters:

- **SchedulingTimeout** `(int: <optional>)`: Defines the maximum duration (in seconds) that a task is permitted to continue to attempt to be scheduled. If all the executions for the task have not been scheduled within this time, the task will fail.

- **BaseRetryDelay** `(int: <optional>)`: Defines the time (in seconds) to wait before trying again if the initial attempt to schedule the job fails.

- **RetryDelayGrowthFactor** `(float: <optional>)`: Defines the factor by which the delay before retrying the scheduling of a job increases after each attempt. A factor of 1.0 will mean that it always waits for the **BaseRetryDelay**; a factor of 1.5 will mean that the delay increases by 50% each time; and a factor of 0.4 will mean that the delay halves each time.

- **MaximumRetryDelay** `(int: <optional>)`: Defines the maximum time (in secoonds) to wait before retrying again, so that an exponential growth caused by a large **RetryDelayGrowthFactor** can be capped at a sensible limit.

Utilizing the `ReschedulingPolicy` allows you to customise the behaviour of jobs that cannot be scheduled due to limited capacity in the cluster. Lower retry delays and growth factors will cause more aggressive attempts to retry scheduling, causing jobs to be scheduled sooner when capacity becomes available, but will increase the load on the system due to repeated failed schedulings in the meantime.

The default **SchedulingTimeout** of one minute means that jobs which will not start quickly will not hang around for a while; if you are submitting a large number of jobs compared to the size of your cluster, so you expect them to have to wait to get a turn to execute, you should set this to a number larger than the expected running time of your batch of jobs.
