---
sidebar_label: 'Image Processing'
sidebar_position: 3
---

# Image Processing

## Introduction

Often, you will need to process a number of images across an entire set hosted on IPFS. For example, the entire [Landsat data dataset is hosted on IPFS](https://ipfs.io/ipfs/QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72). This is many thousands of images, it would be very convenient to run a job against the data without having to download it!

## Getting Started

In this example we will be working against a small [subset of the dataset](https://ipfs.io/ipfs/QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72). We will go through a demo similar to what you may need to do at scale: resizing all the images down to 100x100px.

To get started with a new concept, `bacalhau docker run` takes a `-v` argument just like Docker, except the left hand side of the argument is a CID. This is a *data volume* and can mount in an entire directory (instead of a single file).

When you set this flag, it then ensures that CID is mounted into the container at the desired location as an input volume.

Data volumes also work on output - `bacalhau docker run` also supports a `-o` argument for output volumes. This is where you write the results of your job. See below for an example.

```bash
bacalhau docker run \
  -v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images \
  dpokidov/imagemagick:7.1.0-47-ubuntu \
  -- magick mogrify -resize 100x100 -quality 100 -path /outputs '/input_images/*.jpg'
```

```bash
bacalhau describe JOB_ID | yq .Shards
```

Replace `JOB_ID` with the first part of the job id from the last step.

```bash
- ShardIndex: 0
  Nodes:
    - Node: QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF
      State: Cancelled
      Status: ""
      Verified: false
      ResultID: ""
    - Node: QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
      State: Completed
      Status: 'Got results proposal of length: 0'
      Verified: true
      ResultID: bafybeidtitnyfotvcxa2tu7zjbe6a5mi4q4nkveiu5bm3zeyrzvt4fs7na
```

Since the job state is complete, the job result can be downloaded using

```bash
bacalhau get JOB_ID
```
