---
sidebar_label: 'Installation' sidebar_position: 2
---

# Getting Started with the Public Bacalhau Network

## Install the CLI

```
curl -sL https://bacalhau.org/install.sh | bash
```

## Run a "hello world" job

```
bacalhau run ubuntu echo hello
```

```
[2207] INF jsonrpc/jsonrpc_client.go:73 > Submitted Job Id: fa11069f-17e0-47de-b8b5-37444cb396b5
fa11069f-17e0-47de-b8b5-37444cb396b5
```
Copy the first part of the job id, such as `fa11069f`, we'll use it in a moment.

## Check the status of the job

```
bacalhau list --wide |grep -A 2 JOB_ID
```

Replace `JOB_ID` with the job id you copied in the last step.

Only one of the production servers accepted the job, because you only requested a concurrency setting of 1 (the default in `bacalhau run`).

You should see something like
```
 fa11069f   QmdZQ7Zb  bid_rejected  /ipfs/
            QmXaXu9N  complete      /ipfs/QmQtZKRPXehLU5JroBbzBCVdhNkgZT7m4MiSD7sUVxE3LD
            QmYgxZiy  bid_rejected  /ipfs/
```

Copy the CID (in this case `QmQtZKRPXehLU5JroBbzBCVdhNkgZT7m4MiSD7sUVxE3LD`), and run:

```
ipfs get CID
```
Replace `CID` with the CID you copied above.

```
Saving file(s) to QmQtZKRPXehLU5JroBbzBCVdhNkgZT7m4MiSD7sUVxE3LD
 120 B / 120 B [==================================================================] 100.00% 0s
```

Now read the stdout
```
cat QmQtZKRPXehLU5JroBbzBCVdhNkgZT7m4MiSD7sUVxE3LD/stdout
```

```
hello
```

Hooray, you have just run a job on the Bacalhau network!


# Introducing data and volumes

We are hosting some [Landsat data on IPFS](http://cloudflare-ipfs.com/ipfs/QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72) on the production Bacalhau nodes.

You can run a job against the data without having to download it!
In this example we resize all the images down to 100x100px.

`bacalhau run` takes a `-v` argument just like Docker, except the left hand side of the argument is a CID.
It then ensures that CID is mounted into the container at the desired location as an input volume.

`bacalhau run` also supports a `-o` argument for output volumes. This is where you write the results of your job. See below for an example.

```
bacalhau run \
  -v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images \
  -o results:/output_images \
  dpokidov/imagemagick \
  -- magick mogrify -resize 100x100 -quality 100 -path /output_images /input_images/*.jpg
```

```
bacalhau list |grep -A 2 JOB_ID
```
Replace `JOB_ID` with the first part of the job id from the last step.

```
 ID        JOB          INPUTS  OUTPUTS  CONCURRENCY  NODE      STATE         RESULT
 eb9e5f9e  docker d...       1        1               QmdZQ7Zb  complete      /ipfs/QmWngMTGcn4rM81ePQjMvAEm7rMT4brWh2DXTxD71Le532
```
Go look at the [output](http://cloudflare-ipfs.com/ipfs/QmWngMTGcn4rM81ePQjMvAEm7rMT4brWh2DXTxD71Le532)!


If you would like us to pin some other data you want to run processing jobs against, come ask us in the #bacalhau channel on the [Filecoin slack](https://filecoin.io/slack)!


# Development

You can run an entire Bacalhau + IPFS network on your laptop with the following guide.

The easiest way to spin up bacalhau and run a fully local demo is to use the devstack command. Please see [Running Locally](https://github.com/filecoin-project/bacalhau/blob/main/docs/running_locally.md) for instructions.
