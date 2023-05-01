---
sidebar_label: 'Debugging Jobs'
sidebar_position: 1
description: 'How to troubleshoot and debug failed Bacalhau jobs'
---

# Debugging Failed Jobs

> "An expert is a person who has made all the mistakes that can be made in a very narrow field." ― Niels Bohr

Bacalhau is a decentralized compute network that anyone can join. The network comprises of a smorgasbord of hardware provided by a hodgepodge of providers. In addition, its users are diverse and their jobs are unique. The permutations involved mean that there's a pretty good chance that something will go wrong at some point.

Being decentralized also means that you can't follow standard debugging practices such as SSH'ing into a node or spinning up a REPL environment. This page describes a few hints and tips that we've found useful when debugging failed jobs.

## 1. What Does a Job Failure Look Like?

A failing job could be described as anything that didn't meet your expectations. But clearly much of that is outside of the scope of the network.

When it comes to Bacalhau, a failing job is one that has failed to complete successfully. If you run a job in the foreground you might see a message saying:

```
Error while executing the job.
```

Or when you list the jobs you might see a state of `ERROR`, like:

```
CREATED   ID        JOB                      STATE      VERIFIED  PUBLISHED               
11:05:47  bab5f64c  Docker ubuntu echo "...  Error    
```

## 2. Inspecting the Status of the Job

When you first suspect that your job has failed, the first thing you should do is inspect the status. The `bacalhau describe $JOB_ID` command presents everything that is known about a job from the perspective of the network.

Look through the `Shards` of the job and see if any of them have a `State` of `Error`. The `RunOutput` field provides the juicy details of what went wrong.

## 3. Common Error 1 - `Executable file not found`

One of the most common reasons for failure is that the entrypoint for a job doesn't exist. The `stderr` or `runnerError` will look something like:

```
JobState:
  Nodes:
    QmXMzb3GQRMyUyVvUB53nfkZ1sURTVxuR8BPowey7a3WKk:
      Shards:
        "0":
          NodeId: QmXMzb3GQRMyUyVvUB53nfkZ1sURTVxuR8BPowey7a3WKk
          PublishedResults: {}
          RunOutput:
            exitCode: 0
            runnerError: 'Executable file not found: Error response from daemon: failed
              to create shim task: OCI runtime create failed: runc create failed:
              unable to start container process: exec: "echo \"Something spooky\"
              &>2 && exit 1": executable file not found in $PATH: unknown: Executable
              file not found: Error response from daemon: failed to create shim task:
              OCI runtime create failed: runc create failed: unable to start container
              process: exec: "echo \"Something spooky\" &>2 && exit 1": executable
              file not found in $PATH: unknown'
```

This is usually caused by a mistake in the path to the executable or quotes. To fix this, you'll need to edit the command and make sure it's a valid command.

:::tip
Try enclosing your command in a bash -c '...' (or equivalent shell) to make sure that your command is parsed by the process in the container, not your shell.
:::

## 4. Common Error 2 - `exit code was not zero: 1`

If your program exits with a non-zero exit code, the job will report a failure. The `exitCode` field will present the code. Inspect the `stderr` or `stdout` to see what went wrong. For example:

```
JobState:
  Nodes:
    QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG:
      Shards:
        "0":
          NodeId: QmVAb7r2pKWCuyLpYWoZr9syhhFnTWeFaByHdb8PkkhLQG
          PublishedResults: {}
          RunOutput:
            exitCode: 1
            runnerError: 'exit code was not zero: 1'
            stderr: |
              Something spooky happened and I got scared
```

Typically this is caused by a user error in the code. But you can sometimes see it due to a hardware (e.g. an out of memory error) or Docker error (e.g. wrong container architecture).

## 5. Debugging Via Sanity Checks

If you're not sure what went wrong, you can try adding some sanity checks to your command or code. Here is a list of common command line commands that we use to make sure everything is in its right place:

* `ls -lah /inputs > /outputs/ls.txt` - list the contents of a directory and write to the outputs (or stdout) to double check that files/binaries really exist
* `md5sum /inputs/data.tar.gz > /outputs/checksum.txt` - calculate the checksum of a file and write to the outputs (or stdout) to double check that the file is what you expect

Inside your code:

* Use your language's assert functionality to check that the inputs are what you expect


:::info
Seriously, we've seen all sorts of wonderful things go wrong. Like
CIDs presenting a corrupted file. It's worth checking everything!
:::

More tips:

* [Wikipedia](https://en.wikipedia.org/wiki/Debugging)

## 6. Debugging Via Logging

> "The most effective debugging tool is still careful thought, coupled with judiciously placed print statements." — Brian Kernighan, "Unix for Beginners" (1979)

Since Bacalhau jobs have no external access, you can't rely on remote metric solutions or writing checkpoints to disk. Instead, liberally apply print statements like you're decorating a 1970's Christmas tree.

At the command line:

* `cp /inputs/data.tar.gz /outputs/data.tar.gz` - copy a file to the outputs so you can download and inspect it later
* Add `echo` or `cat` commands to list out pertinent information

Inside your code:

* Use a logging framework if you have one - structure the output to make it more searchable
* Add `print`-like debugging statements to trace the path of execution within your code. When you think you've added enough, add more.
* `print` out the hardware resources observed by your code, to ensure that hardware is visible and behaving as expected (e.g. GPU information)
* For longer-running or hardware intensive jobs, `print` status updates and current consumption metrics to ensure that the job is progressing as expected

More tips: 

* [Tips for debugging with print()](https://adamj.eu/tech/2021/10/08/tips-for-debugging-with-print/)
* [Debugging: print statements and logging](https://firstmncsa.org/2018/12/09/debugging-print-statements-and-logging/)
* [Flame war: The unreasonable effectiveness of print debugging](https://news.ycombinator.com/item?id=26925570)

## 8. Debugging by Running Locally

It might sound obvious but run a test job locally first. You'll often have much better visibility into what's going on and you can use your local tools to debug.

## 7. Debugging Via Simple Jobs

Before running a Bacalhau job for real, it's worth taking the time to slowly baby-step your way to the final command. This is especially true if you're new to Bacalhau or if you're not sure what the inputs will look like.

* Your first job should be a simple `ubuntu` based `ls` command to make sure that the inputs are where you expect them to be
* Your second job should be a similarly simple `ls`-like job, but using your code/container
* Your third job should use your code, but run some kind of "inspect" or "list" or "sanity check" like job to double check that your job has everything it needs to do before it starts. A "hello world" if you will.
* Finally, try and run the actual job.
* If the job fails, try to tailor a job that tests the specific issue you're facing.

In essence, you should try and derisk the job by intentionally testing all the normal things that can go wrong, like data not being in the right place or in the wrong format.

## Support

If you're still having trouble, please reach out to the [Bacalhau team via Slack (#bacalhau channel)](https://join.slack.com/t/bacalhauproject/shared_invite/zt-1sihp4vxf-TjkbXz6JRQpg2AhetPzYYQ)

## Contributing

If you have any hints or tips to add, then please submit a PR to [the Bacalhau Documentation repository](https://github.com/bacalhau-project/docs.bacalhau.org/).

