# How to Release Bacalhau

Two major things that need releasing: the CLI and the production Bacalhau network (a.k.a. the `mainnet`). If you release a new CLI, you must also update the operational networks.

## Creating a CLI Release

**TL;DR**: Create a release on the Github release page and wait for binaries to be built.

> _Note: You only need to create a CLI release if you want to release new Bacalhau functionality. You don't need to cut a release if you are only updating the infrastructure, like altering VM sizes, for example._

1. [Draft a new release](https://github.com/bacalhau-project/bacalhau/releases/new)
2. Create a new tag using semantic versioning, prefixed with a v, postfixed with a `-anything`. E.g. `v0.1.37-alpha1`. The `-xxx` is important because it prevents Docker and Go "latest" issues.
3. Make the title of the release the same as the tag
4. Click on the `Generate Release Notes` button to auto-populate the notes. Add anything else.
5. Tick the "This is a pre-release" checkbox. This prevents this version from being installed by the `https://get.bacalhau.org/install.sh` script.
6. Click on the publish release button. **The [CI scripts](../.circleci) will automatically build and attach binaries.**
7. Download the binaries and test that they do what you expect. Use the pre-release option to download the newest pre-release version. `(export PRE_RELEASE=true ; curl -sL https://get.bacalhau.org/install.sh | bash)` If you are testing against a dev/staging cluster, you **must** update the ops deployments to make the new version work because of signature errors. See [troubleshooting below](#hints-tips-and-troubleshooting).
8. Let the pre-release soak for a week. [Update the Bacalhau Staging servers -- see below](#updating-the-bacalhau-networks). If no issues are found, proceed to the next step.
9. Go back and repeat these steps from step 1, but this time specify a final version number. E.g. `v0.1.37`.
10. Edit the release and de-select the "This is a pre-release" checkbox. You must also re-tick the `Set as the latest release` checkbox. Otherwise the get.bacalhau script will not see it as "latest".
11. [Update the Bacalhau servers -- see below](#updating-the-bacalhau-networks).
12. Inform the head of DevRel (Iryna) so we can write a blog post!

## Updating the Bacalhau Networks

**TL;DR**: Manually deploy the development, staging, and production networks. [See the docs](../ops/README.md).

There are three environments: development, staging and production.

1. Do your infrastructure development (if any) in the development environment.
1. When ready, create a Bacalhau release. Once the release has been built, CI will open a new PR to update the terraform files. Test this change in development manually if you wish. Once you are happy, merge the PR.
1. [Manually apply the changes to the development, staging, and production environment.](../ops/README.md#deploying-bacalhau-mainnet) Please note that it takes a couple of minutes for the init scripts to install and start the Bacalhau servers. You can see what the server is doing with `gcloud compute ssh bacalhau-vm-$WORKSPACE-0 -- journalctl -f`.
1. Make sure to update the [Monitoring Canary as well](../ops/aws/canary/README.md#releasing-a-new-version).

## Hints, Tips and Troubleshooting

### Testing on the Development/Staging Cluster

1. Find out the IP addresses of the cluster from the terraform output
1. Use the `api-host` and `api-port` variables to connect to the cluster. E.g. `bacalhau --api-port=1234 --api-host=35.204.4.66 get 1f68d212-98b6-4cda-aa42-c888a6e834d5`

### Error: publicapi: received non-200 status: 400 client's signature is invalid

This happens when the ops cluster versions are _behind_ the CLI versions. Update the ops clusters to the most recent version.

### Split Brain

When making changes that only affect "node 0" in the network, you will effectively create a new Bacalhau network, and old nodes (1+) will still belong to the old network.

This has the effect of causing issues from a user perspective because each CLI request is being load balanced to two different clusters.

To fix this you must taint all of the old nodes in the network and force terraform to recreate them. E.g. `terraform taint "google_compute_instance.bacalhau_vm[1]"`

### Server Logs

You can see the server logs with:

```bash
gcloud compute ssh bacalhau-vm-$WORKSPACE-0 -- journalctl -f
```

## Further Information

* [Go recommendations for pre-release](https://go.dev/blog/publishing-go-modules#semantic-versions-and-modules)