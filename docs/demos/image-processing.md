---
sidebar_label: 'Image Processing'
sidebar_position: 3
---

# Image Processing 

## Introduction

Often, you will need to process a number of images across an entire set hosted on IPFS. For example, the entire [Landsat data dataset is hosted on IPFS](http://cloudflare-ipfs.com/ipfs/QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72). This is many thousands of images, it would be very convenient to run a job against the data without having to download it!

## Getting Started
In this example we will be working against a small [subset of the dataset]((http://cloudflare-ipfs.com/ipfs/QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72)). We will go through a demo similar to what you may need to do at scale: resizing all the images down to 100x100px.

To get started with a new concept, `bacalhau run` takes a `-v` argument just like Docker, except the left hand side of the argument is a CID. This is a *data volume* and can mount in an entire directory (instead of a single file).

When you set this flag, it then ensures that CID is mounted into the container at the desired location as an input volume.

Data volumes also work on output - `bacalhau run` also supports a `-o` argument for output volumes. This is where you write the results of your job. See below for an example.

```
bacalhau run \
  -v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images \
  -o results:/output_images \
  dpokidov/imagemagick \
  -- magick mogrify -resize 100x100 -quality 100 -path /output_images /input_images/*.jpg
```

```
bacalhau list --sort-by=id --reverse --id-filter= JOB_ID
```
Replace `JOB_ID` with the first part of the job id from the last step.

```
 ID        JOB          INPUTS  OUTPUTS  CONCURRENCY  NODE      STATE         RESULT
 eb9e5f9e  docker d...       1        1               QmdZQ7Zb  complete      /ipfs/QmWngMTGcn4rM81ePQjMvAEm7rMT4brWh2DXTxD71Le532
```
Go look at the [output](http://cloudflare-ipfs.com/ipfs/QmWngMTGcn4rM81ePQjMvAEm7rMT4brWh2DXTxD71Le532)!

