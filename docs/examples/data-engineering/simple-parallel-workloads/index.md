---
sidebar_label: 'Simple Parallel Workloads'
sidebar_position: 2
description: "Parallel Video Resizing via File Sharding"
---
# Parallel Video Resizing via File Sharding

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-engineering/simple-parallel-workloads/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-engineering%2Fsimple-parallel-workloads%2Findex.ipynb)

Many data engineering workloads consist of embarrassingly parallel workloads where you want to run a simple execution on a large number of files. In this example tutorial, we will use the [Bacalhau Sharding](https://docs.bacalhau.org/next-steps/parallel-workloads) to run a simple video filter on a large number of video files.

## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Submit the workload

To submit a workload to Bacalhau, we will use the `bacalhau docker run` command.


```bash
%%bash --out job_id
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

The job has been submitted and Bacalhau has printed out the related job id. We store that in an environment variable so that we can reuse it later on.

The `bacalhau docker run` command allows one to pass input data volume with a `-v CID:path` argument just like Docker, except the left-hand side of the argument is a [content identifier (CID)](https://github.com/multiformats/cid). This results in Bacalhau mounting a *data volume* inside the container. By default, Bacalhau mounts the input volume at the path `/inputs` inside the container.

To shard across files in the input directory, we need to pass three (optional) arguments to the command:

- `sharding-base-path` - the path to the directory you want to shard over
- `sharding-glob-pattern` - the pattern to match files in the directory
- `sharding-batch-size` - the number of files to pass into each job

We created a 72px wide video thumbnails for all the videos in the `inputs` directory. The `outputs` directory will contain the thumbnails for each video. We will shard by 1 video per job, and use the `linuxserver/ffmpeg` container to resize the videos.

:::tip
[Bacalhau overwrites the default entrypoint](https://github.com/bacalhau-project/bacalhau/blob/v0.2.3/cmd/bacalhau/docker_run.go#L64) so we must run the full command after the `--` argument. In this line you will list all of the mp4 files in the `/inputs` directory and execute `ffmpeg` against each instance.
:::


## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
bacalhau list --id-filter=${JOB_ID} --no-style
```

When it says `Published` or `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe ${JOB_ID}
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
mkdir -p ./results # Temporary directory to store the results
bacalhau get --output-dir ./results ${JOB_ID} # Download the results
```

After the download has finished you should see the following contents in results directory.

## Viewing your Job Output

Each job creates 3 subfolders: the **combined_results**,**per_shard files**, and the **raw** directory. To view the file, run the following command:

### Display the videos

To view the videos, we will use **glob** to return all file paths that match a specific pattern.

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

## Need Support?

For questions, feedback, please reach out in our [forum](https://github.com/bacalhau-project/bacalhau/discussions)
