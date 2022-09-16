---
sidebar_label: 'Simple Image Processing'
sidebar_position: 1
---
# Simple Image Processing

In this example we will show you how to use Bacalhau to process images on a Landsat dataset.

If you recall, Bacalhau has the unique capability of operating at massive scale in a distributed environment. This is made possible because data is naturally sharded across the IPFS network amongst many providers. We can take advantage of this to process images in parallel.

However, before we do that, this notebook shows you how to use Bacalhau to process images using a [much smaller subset](https://cloudflare-ipfs.com/ipfs/QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72). This is useful for testing and debugging your code.

For a live walk through of this demo please watch the first part of the video below, otherwise feel free to run the demo yourself by following the steps below.

[![Bacalhau Intro Video](/img/Bacalhau_Intro_Video_thumbnail.jpg)](https://www.youtube.com/watch?v=wkOh05J5qgA)

## Prerequistes

Make sure you have the latest `bacalhau` client installed by following the [getting started instructions](../../../getting-started/installation) or using the installation command below (which installs Bacalhau local to the notebook).


```python
import subprocess
rc = subprocess.call(['command', '-v', 'bacalhau'])
if rc == 0:
    print('bacalhau installed, skipping installation')
else:
    !command -v docker >/dev/null 2>&1 || { echo >&2 "I require docker but it's not installed.  Aborting."; exit 1; }
    !(export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
    path=!echo $PATH
    %env PATH=./:{path[0]}
```

    Your system is darwin_arm64
    
    BACALHAU CLI is detected:
    Client Version: v0.2.3
    Server Version: v0.2.3
    Reinstalling BACALHAU CLI - ./bacalhau...
    Getting the latest BACALHAU CLI...
    Installing v0.2.3 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.2.3/bacalhau_v0.2.3_darwin_arm64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.2.3/bacalhau_v0.2.3_darwin_arm64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into . successfully.
    Client Version: v0.2.3
    Server Version: v0.2.3
    env: PATH=./:/Users/phil/.pyenv/versions/3.9.7/bin:/opt/homebrew/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/bin:/Users/phil/.gvm/bin:/opt/homebrew/opt/findutils/libexec/gnubin:/opt/homebrew/opt/coreutils/libexec/gnubin:/opt/homebrew/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/bin:/Users/phil/.pyenv/shims:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/MacGPG2/bin:/Users/phil/.nexustools



```bash
%%bash
bacalhau version
```

    Client Version: v0.2.3
    Server Version: v0.2.3


## Submit the workload

To get started with a new concept, `bacalhau docker run` allows you to pass input data volume with a `-v CID:path` argument just like Docker, except the left hand side of the argument is a [content identifier (CID)](https://github.com/multiformats/cid).
This results in a *data volume* and can mount in an entire directory (instead of a single file).

When you set this flag, it then ensures that CID is mounted into the container at the `path` location as an input volume.

Data volumes also work on output - by default `bacalhau docker run` always creates an output data volume mounted at `/outputs`.
This is a convenient location to store the results of your job. See below for an example.


```bash
%%bash --out job_id
bacalhau docker run \
  --wait \
  --wait-timeout-secs 100 \
  -v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images \
  dpokidov/imagemagick:7.1.0-47-ubuntu \
  -- magick mogrify -resize 100x100 -quality 100 -path /outputs '/input_images/*.jpg'
```


```python
%env JOB_ID={job_id}
```

    env: JOB_ID=7707ddca-1d83-45db-bcda-0134bdb6a46b


The job has been submitted and Bacalhau has printed out the related job id.
We store that in an environment variable so that we can reuse it later on.


```bash
%%bash
bacalhau list --id-filter=${JOB_ID} --no-style
```

     CREATED   ID        JOB                      STATE      VERIFIED  PUBLISHED               
     14:08:54  7707ddca  Docker dpokidov/imag...  Published            /ipfs/bafybeidtitnyf... 


Since the job state is published/complete, the job is ready to be downloaded.

## Get results

First, let us create a new directory that will store our job outputs.
Second, use the `get` verb to download the job outputs into the directory specified by the `--output-dir` argument.
_Please ignore the `> /dev/null 2>&1` portion of the command, it is there only temporarily until we fix this [issue #614](https://github.com/filecoin-project/bacalhau/issues/614) and is meant to supress debug logs that are not useful for the user._


```bash
%%bash
echo ${JOB_ID}
```

    7707ddca-1d83-45db-bcda-0134bdb6a46b



```bash
%%bash
mkdir -p ./results # Temporary directory to store the results
bacalhau get --output-dir ./results ${JOB_ID} # Download the results
```

    [90m15:09:14.745 |[0m [32mINF[0m [1mbacalhau/get.go:67[0m[36m >[0m Fetching results of job '7707ddca-1d83-45db-bcda-0134bdb6a46b'...
    [90m15:09:17.397 |[0m [32mINF[0m [1mipfs/downloader.go:115[0m[36m >[0m Found 1 result shards, downloading to temporary folder.
    [90m15:09:27.487 |[0m [32mINF[0m [1mipfs/downloader.go:195[0m[36m >[0m Combining shard from output volume 'outputs' to final location: '/Users/phil/source/bacalhau-project/examples/data-engineering/image-processing/results'


The docker run command above used the `outputs` volume as a results folder so when we download them they will be stored in a  folder within `volumes/outputs`.


```bash
%%bash
ls -lah results/volumes/outputs
```

    total 192K
    drwxr-xr-x 11 phil staff 352 Sep 16 15:09 .
    drwxr-xr-x  3 phil staff  96 Sep 16 15:09 ..
    -rw-r--r--  1 phil staff 15K Sep 16 15:09 cafires_vir_2021231_lrg.jpg
    -rw-r--r--  1 phil staff 34K Sep 16 15:09 greatsaltlake_oli_2017210_lrg.jpg
    -rw-r--r--  1 phil staff 13K Sep 16 15:09 greecefires_oli_2021222_lrg.jpg
    -rw-r--r--  1 phil staff 17K Sep 16 15:09 haitiearthquake_oli_20212_lrg.jpg
    -rw-r--r--  1 phil staff 42K Sep 16 15:09 iwojima_tmo_2021225_lrg.jpg
    -rw-r--r--  1 phil staff 11K Sep 16 15:09 lakemead_etm_2000220_lrg.jpg
    -rw-r--r--  1 phil staff 14K Sep 16 15:09 lapalma_oli_2021141_lrg.jpg
    -rw-r--r--  1 phil staff 14K Sep 16 15:09 spainfire_oli_2021227_lrg.jpg
    -rw-r--r--  1 phil staff 16K Sep 16 15:09 sulphursprings_oli_2019254_lrg.jpg


## Where to go next?

* [Take a look at other exmaples](..)
* [How to run an existing workload on Bacalhau](../../../getting-started/workload-onboarding).
* [Check out the Bacalhau CLI Reference page](../../../all-flags).

## Support

Please reach out to the [Bacalhau team via Slack](https://filecoinproject.slack.com/archives/C02RLM3JHUY) to seek help or in case of any issues.
