---
sidebar_label: 'Onboard Your Workload' sidebar_position: 2
---
import ReactPlayer from 'react-player'

# Onboarding Your Workloads

## Steps to onboard your workload

### 1. Modify your workload scripts
Modify your workload (scripts) so that any input files are read from a [local directory](https://docs.bacalhau.org/about-bacalhau/architecture#input--output-volumes) within the Docker container. All ingres/egres networking is disabled from the Bacalhau cluster.

### 2. Build the docker container
Build a an **x86_64 / amd64** based docker image for your workload ([example here](https://docs.docker.com/language/python/build-images/)) and push the image to a [public docker registry](https://codefresh.io/docs/docs/integrations/docker-registries/). Please note: do not build your docker image on a arm64 (Apple Silicon) Mac, the Bacalhau testnet is running x86_64 servers, so the docker images must be built on the same CPU architecture. You may execute bacalhau jobs from the CLI on a Mac, but please avoid building your docker images there.


### 3. Test the docker image locally
Executing the following style of command to test your docker image locally:

```
docker run -v /host-mount-location:/container-input-location/  \
 -o output-folder-name:/container-output-location/
IMAGENAME
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

Here is an [example of an onboarded workload leveraging the Surface Ocean CO₂ Atlas (SOCAT)](https://github.com/wesfloyd/bacalhau_socat_test) to Bacalhau: 

<ReactPlayer playing controls url='https://www.youtube.com/watch?v=t2AHD8yJhLY' playing='false'/>



## Support

Please reach out to the [Bacalhau team via Slack](https://filecoinproject.slack.com/archives/C02RLM3JHUY) if you would like help pinning data to IPFS for your job or in case of any issues.
