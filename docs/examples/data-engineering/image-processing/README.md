---
sidebar_label: 'Image Processing'
sidebar_position: 1
jupyter:
  jupytext:
    notebook_metadata_filter: sidebar_label,sidebar_position
    text_representation:
      extension: .md
      format_name: markdown
      format_version: '1.3'
      jupytext_version: 1.14.1
  kernelspec:
    display_name: Python 3
    language: python
    name: python3
---

<!-- #region -->
# Image Processing


## Introduction

Often, you will need to process a number of images across an entire data set hosted on IPFS. For example, the entire [Landsat data dataset is hosted on IPFS ](https://ipfs.io/ipfs/QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72). This is many thousands of images, it would be very convenient to run a job against the data without having to download it!

This page is a demo of a data intensive image processing workload run on Bacalhau that transforms very high resolution imagery into thumbnail-size pictures.
It is an example of a highly parellizable compute task where a resize function is applied over a large number of files.
For a live walk through of this demo please watch the first part of the video below, otherwise feel free to run the demo yourself by following the steps below.

[![Bacalhau_Intro_Video.jpg](attachment:d25ced22-7753-474b-b65d-f5b2931830ab.jpg)](https://www.youtube.com/watch?v=wkOh05J5qgA)

<!-- [![image](./Bacalhau_Intro_Video.jpg)](https://www.youtube.com/watch?v=wkOh05J5qgA) -->
<!-- #endregion -->

## Prerequistes

Make sure you have the latest `bacalhau` client installed by following the [getting started instructions](getting-started/installation).

```python
!bacalhau version
```

## Submit the workload

In this example we will be working against a small [subset of the dataset](https://ipfs.io/ipfs/QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72). We will go through a demo similar to what you may need to do at scale: resizing all the images down to 100x100px.

To get started with a new concept, `bacalhau docker run` allows you to pass input data volume with a `-v CID:path` argument just like Docker, except the left hand side of the argument is a [content identifier (CID)](https://github.com/multiformats/cid).
This results in a *data volume* and can mount in an entire directory (instead of a single file).

When you set this flag, it then ensures that CID is mounted into the container at the `path` location as an input volume.

Data volumes also work on output - by default `bacalhau docker run` always creates an output data volume mounted at `/outputs`.
This is a convenient location to store the results of your job. See below for an example.

```python
!bacalhau docker run \
  -v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images \
  dpokidov/imagemagick:7.1.0-47-ubuntu \
  -- magick mogrify -resize 100x100 -quality 100 -path /outputs '/input_images/*.jpg'
```

The job has been submitted and Bacalhau has printed out the related job id.
We store that in an environment variable so that we can reuse it later on.

```python
%env JOB_ID=9536e23c
!bacalhau list --id-filter=${JOB_ID}
```

Since the job state is published/complete, the job result can be downloaded locally.
We achieve that in the next section.


## Get results

First, let us create a new directory that will store our job outputs.
Second, use the `get` verb to download the job outputs into the current directory.
_This command prints out a number of verbose logs, although these meant for Bacalhau developers you may want to ignore them (this will soon: [issue #614](https://github.com/filecoin-project/bacalhau/issues/614))._

```python
!mkdir -p /tmp/img-demo
!bacalhau get ${JOB_ID} --output-dir /tmp/img-demo
```

Now, the docker run command above used the `outputs` volume as a results folder so when we download them they will be stored in a homonymous folder within `volumes/`.

```python
ls -l /tmp/img-demo/volumes/outputs/
```

## Where to go next?

* [How to run an existing workload on Bacalhau](../../../getting-started/workload-onboarding).
* [Check out the Bacalhau CLI Reference page](../../../all-flags).


## Support

Please reach out to the [Bacalhau team via Slack](https://filecoinproject.slack.com/archives/C02RLM3JHUY) to seek help or in case of any issues.

```python

```
