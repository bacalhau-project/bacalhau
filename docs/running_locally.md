# Running locally with the 'devstack' command

The `devstack` command of bacalhau will start a 3 node cluster alongside isolated ipfs servers.

This is useful to kick the tires and/or developing on the codebase.  It's also the tool used by some of the tests.

## Pre-requisites

 * x86_64 of ARM64 architecture
    * Ubuntu 20.0+ has most often been used for development and testing
 * Go >= 1.18
 * [Docker Engine](https://docs.docker.com/get-docker/)
 * (Optional) A build of the [latest Bacalhau release](https://github.com/filecoin-project/bacalhau/releases/)

## (Optional) Building Bacalhau from source

```bash
sudo apt-get update && sudo apt-get install -y make gcc zip
sudo snap install go --classic
wget https://github.com/filecoin-project/bacalhau/archive/refs/heads/main.zip
unzip main.zip
cd bacalhau-main
go build
```

## Start the cluster

```bash
./bin/<YOUR_ARCHITECTURE>/bacalhau devstack
```

This will start a 3 node bacalhau cluster connected with libp2p.

Each node has it's own ipfs server isolated using the `IPFS_PATH` environment variable and it's own API RPC server isolated using a random port.

If you would like to make it a bit more predictable and/or ignore errors (such as during CI), you can add the following before your execution:
```
IGNORE_PID_AND_PORT_FILES=true PREDICTABLE_API_PORT=1
```

Once everything has started up - you will see output like the following:

```bash
-------------------------------
node 0
-------------------------------

export BACALHAU_IPFS_API_PORT_0=62403
export BACALHAU_IPFS_PATH_0=/var/folders/38/kszkvfx157q2qy4fm0gwzxpw0000gn/T/ipfs-tmp2419155176
export BACALHAU_API_HOST_0=0.0.0.0
export BACALHAU_API_PORT_0=62406
cid=$(ipfs --api /ip4/127.0.0.1/tcp/62403 add --quiet ./testdata/grep_file.txt)
curl -XPOST http://127.0.0.1:62403/api/v0/id

-----------------------------------------
-----------------------------------------

export BACALHAU_IPFS_PATH_0=/var/folders/38/kszkvfx157q2qy4fm0gwzxpw0000gn/T/ipfs-tmp2419155176
export BACALHAU_API_HOST_0=0.0.0.0
export BACALHAU_API_PORT_0=62406
export BACALHAU_API_HOST=0.0.0.0
export BACALHAU_API_PORT=62406
```

The last two lines contain the environment variables you need for a new window.

## New Terminal Window
* Open an additional terminal window to be used for submitting jobs.
* Copy and paste the last two lines into this window. EG:

```bash
export BACALHAU_API_HOST=0.0.0.0
export BACALHAU_API_PORT=62406
```
You are now ready to submit a job to your local devstack.

## Submit a simple job

This will submit a simple job to a single node:

```bash
go run . docker run ubuntu echo "hello devstack test"
```

This should output something like the following:
```bash
15:01:00.638 | INF bacalhau/utils.go:69 > Development client version, skipping version check
d7d4d23d-08ff-46f4-a695-f37647da67cc
```

After a short while - the job should be in `complete` state.

```bash
go run . list --wide
 CREATION_TIME      ID                                    JOB                             STATE      RESULT
 22-08-29-15:01:00  d7d4d23d-08ff-46f4-a695-f37647da67cc  Docker ubuntu echo hello world  Published  /ipfs/QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf
```

Download the results to the current directory:
```bash
go run . get d7d4d23d-08ff-46f4-a695-f37647da67cc # Works with partial IDs - just the first 8 characters
```

You should now have the following files and directories:
- stdout
- stderr
- volumes/output

If you `cat stdout` it should read "hello devstack test". If you write any files in your job, they will appear in volumes/output.
