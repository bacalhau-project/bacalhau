# Benchmark hack

## Run Bacalhau Nodes
```bash
export PREDICTABLE_API_PORT=1 # to force nodes start with port 20000
make devstack # runs 3 nodes by default
```
or to test without actually running docker jobs

```bash
make devstack-noop
```

or to test with *n* number of nodes
```bash
go run . devstack --compute-nodes <n>
```

## Generate Requests

copy paste export commands, e.g.
```bash
export API_PORT=20000 # requester node's port
export BACALHAU_BIN=$(which bacalhau) # cli used to send requests
```

**Test one job**

```bash
cd benchmark
bash submit.sh
```

**Test multiple jobs**

```bash
bash explode.sh
```

You also have the following configurations for multiple jobs
```bash
export TOTAL_JOBS=60 # Total number of jobs
export BATCH_SIZE=10 # No. of jobs to send sequentially as a single hyperfine run
export CONCURRENCY=2 # No. of concurrent batchs
export REQUESTER_NODES=1 # No. of requester nodes to call
```
In the above example, we have a total of 60 jobs split across 6 (60/10) separate benchmarks. There can only be 2 concurrent benchamrks at a given time, and both will call 2 separate requester nodes.


### FAQ
**I am getting "Too many open files" when trying to run multiple nodes in a single machine!**
Increase your OS `ulimit` to some large value using:
```bash
ulimit -n <large_value> # e.g. 1048576
 ```
