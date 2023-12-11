---
sidebar_label: 'Resource Limits'
sidebar_position: 150
---

# Resource Limits

These are the flags that control the capacity of the Bacalhau node, and the limits for jobs that might be run.

```
  --limit-job-cpu string                 Job CPU core limit for single job (e.g. 500m, 2, 8).
  --limit-job-gpu string                 Job GPU limit for single job (e.g. 1, 2, or 8).
  --limit-job-memory string              Job Memory limit for single job  (e.g. 500Mb, 2Gb, 8Gb).
  --limit-total-cpu string               Total CPU core limit to run all jobs (e.g. 500m, 2, 8).
  --limit-total-gpu string               Total GPU limit to run all jobs (e.g. 1, 2, or 8).
  --limit-total-memory string            Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).
```

The `--limit-total-*` flags control the total system resources you want to give to the network.  If left blank, the system will attempt to detect these values automatically.

The `--limit-job-*` flags control the maximum amount of resources a single job can consume for it to be selected for execution.

Resource limits are not supported for Docker jobs running on Windows. Resource
limits will be applied at the job bid stage based on reported job requirements
but will be silently unenforced. Jobs will be able to access as many resources
as requested at runtime.
