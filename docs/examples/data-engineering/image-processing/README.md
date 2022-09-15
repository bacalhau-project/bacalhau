---
sidebar_label: 'Image Processing'
sidebar_position: 1
---
# Image Processing


## Introduction

Often, you will need to process a number of images across an entire data set hosted on IPFS. For example, the entire [Landsat data dataset is hosted on IPFS ](https://ipfs.io/ipfs/QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72). This is many thousands of images, it would be very convenient to run a job against the data without having to download it!

This page is a demo of a data intensive image processing workload run on Bacalhau that transforms very high resolution imagery into thumbnail-size pictures.
It is an example of a highly parallelizable compute task where a resize function is applied over a large number of files.
For a live walk through of this demo please watch the first part of the video below, otherwise feel free to run the demo yourself by following the steps below.

[![Bacalhau Intro Video](/img/Bacalhau_Intro_Video_thumbnail.jpg)](https://www.youtube.com/watch?v=wkOh05J5qgA)

## Prerequistes

Make sure you have the latest `bacalhau` client installed by following the [getting started instructions](../../../getting-started/installation).


```python
!bacalhau version
```

    Client Version: v0.2.3
    Server Version: v0.2.3


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

    4d49f48a-0522-4016-aa0a-23168d1ca99a


The job has been submitted and Bacalhau has printed out the related job id.
We store that in an environment variable so that we can reuse it later on.


```python
%env JOB_ID=4d49f48a
!bacalhau list --id-filter=${JOB_ID} --no-style
```

    env: JOB_ID=4d49f48a
     CREATED   ID        JOB                      STATE      VERIFIED  PUBLISHED               
     11:33:22  4d49f48a  Docker dpokidov/imag...  Published            /ipfs/bafybeidtitnyf... 


Since the job state is published/complete, the job result can be downloaded locally.
We achieve that in the next section.

## Get results

First, let us create a new directory that will store our job outputs.
Second, use the `get` verb to download the job outputs into the directory specified by the `--output-dir` argument.
_Please ignore the `> /dev/null 2>&1` portion of the command, it is there only temporarily until we fix this [issue #614](https://github.com/filecoin-project/bacalhau/issues/614) and is meant to supress debug logs that are not useful for the user._


```python
!mkdir -p /tmp/img-demo
!bacalhau get ${JOB_ID} --output-dir /tmp/img-demo > /dev/null 2>&1
```

Now, the docker run command above used the `outputs` volume as a results folder so when we download them they will be stored in a homonymous folder within `volumes/`.


```python
ls -l /tmp/img-demo/volumes/outputs/
```

    total 384
    -rw-r--r--  1 enricorotundo  staff  14536 Sep 15 13:42 cafires_vir_2021231_lrg.jpg
    -rw-r--r--  1 enricorotundo  staff  34594 Sep 15 13:42 greatsaltlake_oli_2017210_lrg.jpg
    -rw-r--r--  1 enricorotundo  staff  12928 Sep 15 13:42 greecefires_oli_2021222_lrg.jpg
    -rw-r--r--  1 enricorotundo  staff  16705 Sep 15 13:42 haitiearthquake_oli_20212_lrg.jpg
    -rw-r--r--  1 enricorotundo  staff  42427 Sep 15 13:42 iwojima_tmo_2021225_lrg.jpg
    -rw-r--r--  1 enricorotundo  staff  10419 Sep 15 13:42 lakemead_etm_2000220_lrg.jpg
    -rw-r--r--  1 enricorotundo  staff  13467 Sep 15 13:42 lapalma_oli_2021141_lrg.jpg
    -rw-r--r--  1 enricorotundo  staff  13687 Sep 15 13:42 spainfire_oli_2021227_lrg.jpg
    -rw-r--r--  1 enricorotundo  staff  15476 Sep 15 13:42 sulphursprings_oli_2019254_lrg.jpg


## Where to go next?

* [How to run an existing workload on Bacalhau](../../../getting-started/workload-onboarding).
* [Check out the Bacalhau CLI Reference page](../../../all-flags).

## Support

Please reach out to the [Bacalhau team via Slack](https://filecoinproject.slack.com/archives/C02RLM3JHUY) to seek help or in case of any issues.
