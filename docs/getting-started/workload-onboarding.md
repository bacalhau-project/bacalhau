---
sidebar_label: 'Onboard Your Workload' sidebar_position: 2
---
import ReactPlayer from 'react-player'

# Onboarding Your Workloads

## Steps to onboard your workload

### 1. Modify your workload scripts

#### Inputs

_Note: all ingres/egres networking is disabled from the Bacalhau cluster, which will impact your workload if it pulls input data directly via HTTP._

Option 1) Mount input data folder via Docker mount & IPFS
Use docker mounts for inputs if your data needs to be consumed from IPFS and your workload allows **directory paths** (not individual files) as inputs.
* Modify your workload (scripts) so that any input files are read from a [local directory](https://docs.bacalhau.org/about-bacalhau/architecture#input--output-volumes) mounted to the Docker container.
* Any input files in your script, must be modified to read from files in an "input" folder in your project that can be mounted via IPFS.

Option 2) Mount external HTTP/S URL as a path within Docker container. Per the [Run command CLI flags documentation](https://docs.bacalhau.org/cli-flags/all-flags#run), the ```--input-urls strings``` flag can be used to mount data from an external URL (HTTP/S) to a specified PATH within the Docker container.

Option 3) Embed input data in the Docker image
You can choose to embed your workload's input data within the docker image. As a result, your ```bacalhau docker run``` command will not require an input volume mount from IPFS.

#### Outputs

Modify your workload so that any output files are written to an "output/" folder. This will allow for the clear/specific mounting of the output folder when the "bacalhau docker run" command is executed. 

Please see this [modified script example here](https://github.com/wesfloyd/bacalhau_socat_test/blob/9e51e48d6f9efa4adc8125fe97004c204e387fe5/main.py#L31).


### 2. Build the docker container
Build a an **x86_64 / amd64** based docker image for your workload ([example here](https://docs.docker.com/language/python/build-images/)) and push the image to a [public docker registry](https://codefresh.io/docs/docs/integrations/docker-registries/). 

_Note: do not build your docker image on a arm64 (Apple Silicon) Mac, the Bacalhau testnet is running x86_64 servers, so the docker images must be built on the same CPU architecture. You may execute bacalhau jobs from the CLI on a Mac, but please avoid building your docker images there._


### 3. Test the docker image locally
Executing the following style of command to test your docker image locally:

```
docker run -v /host-mount-location:/container-input-location/  \
  -o output-folder-name:/container-output-location/ \
  IMAGENAME [CMD]
```

### 4. Migrate input data for the workload to IPFS
- Leverage an IPFS “pinning service” such as:
  - [Web3.Storage](https://web3.storage/account/)
  - [Estuary](https://estuary.tech/sign-in)
  - [Manually pin your files to IPFS](https://docs.ipfs.io/how-to/pin-files/) with your own IPFS server.
- If uploading a folder of input files, consider [uploading with this script](https://web3.storage/docs/#create-the-upload-script). However, please note that any content uploaded to Web3.storage is [also wrapped in a parent directory](https://web3.storage/docs/how-tos/store/#directory-wrapping). You will need to take care to reference the inner directory CID in your bacalhau command.


### 5. Run the workload on Bacalhau:

**Note:** Bacalhau does **not** support subpaths within a CID. You must reference the CID of an atomic folder in your `bacalhau docker run` command.
```
bacalhau docker run -v CID:/container-input-location/ \
    -o output-folder-name:/container-output-location/ IMAGENAME

bacalhau list 

bacalhau get JOB_ID
```



## Example Onboarded Workload

Here is an example of an onboarded workload leveraging the Surface Ocean CO₂ Atlas (SOCAT) to Bacalhau:
- [Youtube: Bacalhau SOCAT Workload Demo](https://www.youtube.com/watch?v=t2AHD8yJhLY)
- [Github: bacalhau_socat_test](https://github.com/wesfloyd/bacalhau_socat_test)

<!-- <ReactPlayer playing controls url='https://www.youtube.com/watch?v=t2AHD8yJhLY' playing='false'/> -->

Here is an example of running a job live on the Bacalhau network: [Youtube: Bacalhau Intro Video](https://www.youtube.com/watch?v=wkOh05J5qgA)





## Support

Please reach out to the [Bacalhau team via Slack](https://filecoinproject.slack.com/archives/C02RLM3JHUY) if you would like help pinning data to IPFS for your job or in case of any issues.
