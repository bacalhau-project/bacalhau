---
sidebar_label: 'Simple Parallel Workloads'
sidebar_position: 2
description: "Parallel Video Resizing via File Sharding"
---
# Parallel Video Resizing via File Sharding

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-engineering/simple-parallel-workloads/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-engineering%2Fsimple-parallel-workloads%2Findex.ipynb)

Many data engineering workloads consist of embarrassingly parallel workloads where you want to run a simple execution on a large number of files. In this notebook, we will use the [Sharding](https://docs.bacalhau.org/getting-started/parallel-workloads) functionality in Bacalhau to run a simple video filter on a large number of video files.

> Although you would normally you would use your own container and script to make your workloads reproducible, in this example we will use a pre-built container and CLI arguments to allow you to make changes. You can find the container [on docker hub](https://hub.docker.com/r/linuxserver/ffmpeg).

## Prerequistes

Make sure you have the latest `bacalhau` client installed by following the [getting started instructions](../../../getting-started/installation) or using the installation command below (which installs Bacalhau local to the notebook).

## Submit the workload

To submit a workload to Bacalhau you can use the `bacalhau docker run` command. This allows you to pass input data volume with a `-v CID:path` argument just like Docker, except the left-hand side of the argument is a [content identifier (CID)](https://github.com/multiformats/cid). This results in Bacalhau mounting a *data volume* inside the container. By default, Bacalhau mounts the input volume at the path `/inputs` inside the container.

Bacalhau also mounts a data volume to store output data. By default `bacalhau docker run` creates an output data volume mounted at `/outputs`. This is a convenient location to store the results of your job. See below for an example.

And to shard across files in the input directory, we need to pass three (optional) arguments to the command:

* `sharding-base-path` - the path to the directory you want to shard over
* `sharding-glob-pattern` - the pattern to match files in the directory
* `sharding-batch-size` - the number of files to pass into each job

### A Simple Video Resize Example

In this example, you will create 72px wide video thumbnails for all the videos in the `inputs` directory. The `outputs` directory will contain the thumbnails for each video. We will shard by 1 video per job, and use the `linuxserver/ffmpeg` container to resize the videos.

Note that [Bacalhau overwrites the default entrypoint](https://github.com/filecoin-project/bacalhau/blob/v0.2.3/cmd/bacalhau/docker_run.go#L64) so we must run the full command after the `--` argument. In this line you will list all of the mp4 files in the `/inputs` directory and execute `ffmpeg` against each instance.


```bash
bacalhau docker run \
  --wait \
  --wait-timeout-secs 100 \
  --id-only \
  --sharding-base-path "/inputs" \
  --sharding-glob-pattern "*.mp4" \
  --sharding-batch-size 1 \
  -v Qmd9CBYpdgCLuCKRtKRRggu24H72ZUrGax5A9EYvrbC72j:/inputs \
  linuxserver/ffmpeg -- \
  bash -c 'find /inputs -iname "*.mp4" -printf "%f\n" | xargs -I{} ffmpeg -y -i /inputs/{} -vf "scale=-1:72,setsar=1:1" /outputs/scaled_{}'

```

## Get Results

Now let's download and display the result from the results directory. We can use the `bacalhau results` command to download the results from the output data volume. The `--output-dir` argument specifies the directory to download the results to.


```bash
mkdir -p ./results # Temporary directory to store the results
bacalhau get --output-dir ./results ${JOB_ID} # Download the results
```

<!-- This is for the benefit of the documentation -->
<video src={require('./scaled_Bird_flying_over_the_lake.mp4').default} controls  >
Your browser does not support the <code>video</code> element.
</video>
<video src={require('./scaled_Calm_waves_on_a_rocky_sea_gulf.mp4').default} controls  >
Your browser does not support the <code>video</code> element.
</video>
<video src={require('./scaled_Prominent_Late_Gothic_styled_architecture.mp4').default} controls  >
Your browser does not support the <code>video</code> element.
</video>
