# devstack

The `devstack` command of bacalhau will start a 3 node cluster alongside isolated ipfs servers.

This is useful to kick the tires and/or developing on the codebase.  It's also the tool used by some of the tests.

## pre-requisites

 * go >= 1.16
 * [ipfs cli](https://github.com/ipfs/go-ipfs#install-prebuilt-binaries)
 * [docker](https://docs.docker.com/get-docker/)

## start cluster

```bash
make devstack
```

This will start a 3 node bacalhau cluster connected with libp2p.

Each node has it's own ipfs server isolated using the `IPFS_PATH` environment variable and it's own JSON RPC server isolated using a random port.

Once everything has started up - you will see output like the following:

```bash
-------------------------------                                     
environment                                                         
-------------------------------                                     

IPFS_PATH_0=/tmp/bacalhau-ipfs1110685378                            
JSON_PORT_0=41081                                                   
IPFS_PATH_1=/tmp/bacalhau-ipfs919189468                             
JSON_PORT_1=41057                                                   
IPFS_PATH_2=/tmp/bacalhau-ipfs490124113                             
JSON_PORT_2=41347
```

Copy and paste these variables into another terminal.

## adding files

Each node has it's own `IPFS_PATH` value.  This means you can use the `ipfs` cli in isolation from the other 2 nodes.

For example - to add a file to only one of the ipfs nodes:

```bash
cid=$( IPFS_PATH=$IPFS_PATH_0 ipfs add -q ./testdata/grep_file.txt )
```

This is especially useful when you want to test self selection of a job based on whether the cid is `local` to that node.

## json rpc

Each node is it's own `--jsonrpc-port` value.  This means you can use the `go run .` cli in isolation from the other 2 nodes.

For example - to view the current job list from the perspective of only one of the 3 nodes:

```bash
# replace this with the correct port from the output
export NODE1_JSONRPC_PORT=38967
go run . --jsonrpc-port=$NODE1_JSONRPC_PORT list
```

## run simple job

This will submit a simple job to a single node:

```bash
cid=$( IPFS_PATH=$IPFS_PATH_0 ipfs add -q ./testdata/grep_file.txt )
go run . --jsonrpc-port=$JSON_PORT_0 submit --cids=$cid --commands="grep kiwi /ipfs/$cid"
go run . --jsonrpc-port=$JSON_PORT_0 list
```

After a short while - the job should be in `complete` state.

```
kai@xwing:~/projects/bacalhau$ go run . --jsonrpc-port=$JSON_PORT_0 list
JOB       COMMAND                  DATA                     NODE                     STATE     STATUS                                                               OUTPUT                                         
63b0a80e  grep kiwi /ipfs/QmRy...  QmRyDNzrxwcL4ENNGyKL...  QmcMKp2NQm2QQf7nRFjK...  complete  Got job results cid: QmRZa9mCrjMgMtaaZZTAEBRdHCVJR3WjoncsEuZU9qBpzv  QmRZa9mCrjMgMtaaZZTAEBRdHCVJR3WjoncsEuZU9qBpzv
```

Copy the job id into a variable:

```bash
JOB_ID=63b0a80e
```

Now we can list the results:

```bash
go run . --jsonrpc-port=$JSON_PORT_0 results list $JOB_ID
```

This will show the following:

```
kai@xwing:~/projects/luke/bacalhau$ go run . --jsonrpc-port=$JSON_PORT_0 results list 63b0a80e
NODE                                            IPFS                                                                 RESULTS                                                                      DIFFERENCE  CORRECT 
QmcMKp2NQm2QQf7nRFjKgTaknsdvmFsp4zjJHvAoP9CvRu  https://ipfs.io/ipfs/QmRZa9mCrjMgMtaaZZTAEBRdHCVJR3WjoncsEuZU9qBpzv  ~/.bacalhau/results/63b0a80e/QmcMKp2NQm2QQf7nRFjKgTaknsdvmFsp4zjJHvAoP9CvRu           0  ✅      
```

The results from the job are stored in the `~/.bacalhau/results/<JOB_ID>/<NODE_ID>` directory.

We can see the files that were output by the job here:

```bash
ls -la ~/.bacalhau/results/63b0a80e/QmcMKp2NQm2QQf7nRFjKgTaknsdvmFsp4zjJHvAoP9CvRu
```

## run 3 node job

Now let's run a job across all 3 nodes.  To do this, we need to add the cid to all the IPFS servers so the job will be selected to run across all 3 nodes:

```bash
cid=$( IPFS_PATH=$IPFS_PATH_0 ipfs add -q ./testdata/grep_file.txt )
IPFS_PATH=$IPFS_PATH_1 ipfs add -q ./testdata/grep_file.txt
IPFS_PATH=$IPFS_PATH_2 ipfs add -q ./testdata/grep_file.txt
```

Then we submit the job but with `--concurrency` and `--confidence` settings:

```bash
go run . --jsonrpc-port=$JSON_PORT_0 submit --cids=$cid --commands="grep pear /ipfs/$cid" --concurrency=3 --confidence=2
go run . --jsonrpc-port=$JSON_PORT_0 list
```

We can see that all 3 nodes have produced results by getting the job id and running:

```bash
go run . --jsonrpc-port=$JSON_PORT_0 results list <JOB_ID>
```

## run 3 node job with bad actor

Now let's restart the devstack but this time with one of the three nodes in `bad actor` mode.  This bad node will not run the job and instead just sleep for 10 seconds.

ctrl+c on the running dev-stack and re-run with:

```bash
make devstack-badactor
```

Copy and paste the environment section into your other terminal and then let's submit another job to the 3 nodes:

```bash
cid=$( IPFS_PATH=$IPFS_PATH_0 ipfs add -q ./testdata/grep_file.txt )
IPFS_PATH=$IPFS_PATH_1 ipfs add -q ./testdata/grep_file.txt
IPFS_PATH=$IPFS_PATH_2 ipfs add -q ./testdata/grep_file.txt
go run . --jsonrpc-port=$JSON_PORT_0 submit --cids=$cid --commands="grep pear /ipfs/$cid" --concurrency=3 --confidence=2
go run . --jsonrpc-port=$JSON_PORT_0 list
```

This time - when you list the results, you will see that our bad actor has been caught because their memory trace is substantially different from the others.