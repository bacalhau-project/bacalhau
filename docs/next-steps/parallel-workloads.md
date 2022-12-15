---
sidebar_label: 'Sharding'
sidebar_position: 2
---

# Parallel Workloads

Bacalhau can run workloads in parallel by splitting the input data volumes and running `shards` of the workload on different nodes.

This works by using a **glob pattern** to slice the input data volumes into atoms, then grouping them using a **batch size** into `shards`.

Each shard runs on a different computer, and is executed in parallel. Once all of the shards have completed, the results are merged and the final result is combined from all the shards.

## Glob Pattern

First, you'll need to decide how to split the input data using a glob pattern.

For example, if you have a folder with thousands of images all at the top layer as follows:

 * image00001.jpg
 * image00002.jpg
 * ...
 * image10000.jpg

Then - our glob pattern could be `image*.jpg` which would match 10,000 images.

We might have folders at the top level:

 * folder1/
 * folder2/
 * ...
 * folderN/

For this, our glob pattern could be `folder*`.

## Base Path

The base path is the common path that the glob pattern will be applied to, and can be useful if you combine multiple input volumes into one job.

For example, if we have 10 input volumes each containing a sequence of images as above, then we can mount them all under a single path as follows:

 * /input_images/volume1
   * image00001.jpg
   * image00002.jpg
 * /input_images/volumeX
   * image02001.jpg
   * image02002.jpg
 * /input_images/volume10
   * image04701.jpg
   * image04702.jpg

We can then use a base path of `/input_images` and a glob pattern of `/volume*/image*.jpg`, which would result in all images in all mounted volumes becoming atoms in our shards.

## Batch Size

Once we've have split our into data into atoms based on the glob pattern, we will then combine those atoms into shards using the `batch size`.

In our example of a folder with 10,000 images, we might want to set our batch size to 1000, resulting in 10 shards.

If our workload was image magic, each of the 10 shards would be operating on a different set of input images and would be executed in parallel.

## Putting it together

The following is an example:

```bash
# this CID points at the folder with 10,000 images all named image00001.jpg
export IMAGE_FOLDER_CID=xxx
bacalhau docker run \
  # mount the input folder
  -v $IMAGE_FOLDER_CID:/input_images \
  # name the output folder
  -o results:/output_images \
  # the glob pattern to split the input data into shards
  --sharding-glob-pattern image*.jpg \
  # the base path to start the glob pattern from
  --sharding-base-path /input_images/ \
  # group the atoms into groups of this size
  --sharding-batch-size 1000 \
  # this is the image magic workload that will process all images in a folder
  dpokidov/imagemagick \
    -resize 100x100 -quality 100 \
    -path /output_images \
    /input_images/*.jpg
```
