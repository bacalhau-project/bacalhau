---
sidebar_label: 'Installation' sidebar_position: 1
---
import ReactPlayer from 'react-player'

# Getting Started with the Public Bacalhau Network

## Install the CLI

```
curl -sL https://get.bacalhau.org/install.sh | bash
```

## Run a "hello world" job

```
bacalhau docker run ubuntu echo hello
```

```
[2207] INF jsonrpc/jsonrpc_client.go:73 > Submitted Job Id: fa11069f-17e0-47de-b8b5-37444cb396b5
fa11069f-17e0-47de-b8b5-37444cb396b5
```
Copy the first part of the job id, such as `fa11069f`, we'll use it in a moment.


```
bacalhau list --wide --sort-by=id --id-filter=<JOB_ID>
```

Replace `JOB_ID` with the job id you copied in the last step.

Only one of the production servers accepted the job, because you only requested a concurrency setting of 1 (the default in `bacalhau docker run`).

You should see something like
```
 fa11069f   QmdZQ7Zb  bid_rejected  /ipfs/
            QmXaXu9N  complete      /ipfs/QmQtZKRPXehLU5JroBbzBCVdhNkgZT7m4MiSD7sUVxE3LD
            QmYgxZiy  bid_rejected  /ipfs/
```

Copy the CID (in this case `QmQtZKRPXehLU5JroBbzBCVdhNkgZT7m4MiSD7sUVxE3LD`), and run:

```
bacalhau get CID
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


## Demo Video

Here is an example of running a job live on the Bacalhau network: [Youtube: Bacalhau Intro Video](https://www.youtube.com/watch?v=wkOh05J5qgA)

<!-- <ReactPlayer playing controls url='https://www.youtube.com/watch?v=wkOh05J5qgA' playing='false'/> -->


## Support

Please reach out to the [Bacalhau team via Slack](https://filecoinproject.slack.com/archives/C02RLM3JHUY) if you would like help pinning data to IPFS for your job or in case of any issues.
