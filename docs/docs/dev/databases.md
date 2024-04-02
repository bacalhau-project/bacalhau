# Database Integration

## Requester node database (job store)

Requester nodes store job state and history in a boltdb-backed store (pkg/jobstore/boltdb).

The location of the database file can be specified using the `BACALHAU_JOB_STORE_PATH` environment variable, which will specify which file to use to store the database.  When not specified, the file will be `{$BACALHAU_DIR}/{NODE_ID}-requester.db`.


## Compute node database (execution store)

By default, compute nodes store their execution information in an bolddb-backed store (pkg/compute/store/boltdb).

The location of the database file (for a single node) can be specified using the `BACALHAU_COMPUTE_STORE_PATH` environment variable, which will specify which file to use to store the database.  When not specified, the file will be `{$BACALHAU_DIR}/{NODE_ID}-compute.db`.


### Compute node restarts

As compute nodes restart, they will find they have existing state in the boltdb database.
At startup the database currently iterates the executions to calculate the counters for each state.  This will be a good opportunity to do some compaction of the records in the database, and cleanup items no longer in use.

Currently only batch jobs are possible, and so for each of the listed states below, no action is taken at restart. In future it would make sense to remove records older than a certain age, or moved them to failed, depending on their current state.  For other job types (to be implemented) this may require restarting jobs, resetting jobs,

|State|Batch jobs|
|--|--|
|ExecutionStateCreated| No action |
|ExecutionStateBidAccepted| No action |
|ExecutionStateRunning| No action |
|ExecutionStateWaitingVerification| No action |
|ExecutionStateResultAccepted| No action |
|ExecutionStatePublishing| No action |
|ExecutionStateCompleted| No action |
|ExecutionStateFailed| No action |
|ExecutionStateCancelled| No action |



## Inspecting the databases

The databases can be inspected using the bbolt tool.
The bbolt tool can be installed to $GOBIN with:

```shell
go install go.etcd.io/bbolt/cmd/bbolt
```

Once installed, and assuming the database file is stored in $FILE you can use bbolt to:

### Check the database integrity

```shell
$ bbolt check $FILE
OK
```

### List all buckets

```shell
$ bbolt buckets $FILE
execution
execution-history
execution-index
```

### Compact the database (by copying it)

```shell
$ bolt compact -o DESTINATION_FILE $FILE
262144 -> 262144 bytes (gain=1.00x)
```


### Get some DB stats

```shell
$ bbolt stats $FILE
Aggregate statistics for 3 buckets

Page count statistics
        Number of logical branch pages: 0
        Number of physical branch overflow pages: 0
        Number of logical leaf pages: 3
        Number of physical leaf overflow pages: 0
Tree statistics
        Number of keys/value pairs: 29
        Number of levels in B+tree: 2
Page size utilization
        Bytes allocated for physical branch pages: 0
        Bytes actually used for branch data: 0 (0%)
        Bytes allocated for physical leaf pages: 49152
        Bytes actually used for leaf data: 8991 (18%)
Bucket statistics
        Total number of buckets: 11
        Total number on inlined buckets: 8 (72%)
        Bytes used for inlined buckets: 2743 (30%)
```

### List keys in a bucket

```shell
$ bbolt keys $FILE execution
e-42bdab56-bb27-42ca-a4c2-62aeb0f76342
e-9a937556-ac83-49ec-b794-a94c3aa068b2
e-d81034b8-c7c6-422d-af3f-87d680862a58
e-fb54458c-5bad-496e-a1cb-4fc091d882a1
```

### View a single key

```shell
bbolt get $FILE execution e-9a937556-ac83-49ec-b794-a94c3aa068b2
{"ID":"e-9a937556-ac83-49ec-b794-a94c3aa068b2","Job":{"APIVersion":"V1beta2","Metadata": .... more JSON}
```
