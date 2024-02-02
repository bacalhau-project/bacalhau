---
sidebar_label: 'Resource Limits'
sidebar_position: 150
---

# Resource Limits

When starting a node, you can limit the total amount of resources allocated for executing jobs and the individual amount of resources that can be used by a single job. To do this, use the following flags:

```bash
  --limit-total-cpu string               Total CPU core limit to run all jobs (e.g. 500m, 2, 8).
  --limit-total-gpu string               Total GPU limit to run all jobs (e.g. 1, 2, or 8).
  --limit-total-memory string            Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).

  --limit-job-cpu string                 Job CPU core limit for single job (e.g. 500m, 2, 8).
  --limit-job-gpu string                 Job GPU limit for single job (e.g. 1, 2, or 8).
  --limit-job-memory string              Job Memory limit for single job  (e.g. 500Mb, 2Gb, 8Gb).
```

The `--limit-total-*` flags are responsible for controlling the amount of **total** system resources you want to make available to run jobs. If you do not specify them, the system will automatically determine these values.

The `--limit-job-*` flags set limits for each **single** job, affecting the process of accepting or rejecting jobs.


Resource limits are not supported for Docker jobs running on Windows. Resource
limits will be applied at the job bid stage based on reported job requirements
but will be silently unenforced. Jobs will be able to access as many resources
as requested at runtime.
Also keep in mind that not all resource limits work for WASM. See the [WASM executor description](../../getting-started/resources.md#wasm-executor) for more details.