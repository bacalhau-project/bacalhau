---
sidebar_label: 'Onboard Your Workload' sidebar_position: 2
---

# Onboarding Your Workload

## Migrate your Python workload (script) to Bacalhau

1. Modify your workload (scripts) so that any input files are read from a [local directory](https://docs.bacalhau.org/about-bacalhau/architecture#input--output-volumes) within the Docker container.

2. Build a docker image for your workload ([example here](https://docs.docker.com/language/python/build-images/)) and push the image to a [public docker registry](https://codefresh.io/docs/docs/integrations/docker-registries/).

3. Test the docker image locally by executing:

```
docker run -v /host-mount-location:/container-input-location/  \
 -o output-folder-name:/container-output-location/
IMAGENAME
```

4. Migrate input data for the workload to IPFS. All networking is disabled from within the Bacalhau cluster
Leverage an IPFS “pinning service” such as [Web3.Storage](https://web3.storage/account/), [Estuary](https://estuary.tech/sign-in) or [manually pin the files to IPFS](https://docs.ipfs.io/how-to/pin-files/) with your own IPFS server. If uploading a folder of input files, consider [uploading with this script](https://web3.storage/docs/#create-the-upload-script).

5. Run the workload on Bacalhau:

```
docker run -v /host-mount-location:/container-input-location/ \
    -o output-folder-name:/container-output-location/ IMAGENAME

bacalhau docker run -v CID:/container-input-location/ \
    -o output-folder-name:/container-output-location/ IMAGENAME

bacalhau list

bacalhau get JOB_ID
```

## Example Onboarded Workload
Here is an example of an onboarded python script to Bacalhau: [SOCAT Test](https://github.com/wesfloyd/bacalhau_socat_test)


## Support

Please reach out to the [Bacalhau team via Slack](https://filecoinproject.slack.com/archives/C02RLM3JHUY) in case of any issues.