# How to Release Bacalhau

Two major things that need releasing: the CLI and the production Bacalhau network (a.k.a. the `mainnet`). If you release a new CLI, you must also update the operational networks.

## Creating a CLI Release

**TL;DR**: Create a release on the Github release page and wait for binaries to be built.

> _Note: You only need to create a CLI release if you want to release new Bacalhau functionality. You don't need to cut a release if you are only updating the infrastructure, like altering VM sizes, for example._

1. [Draft a new release](https://github.com/filecoin-project/bacalhau/releases/new)
2. Create a new tag using semantic versioning, prefixed with a v. E.g. `v0.1.37`
3. Make the title of the release the same as the tag
4. Click on the `Generate Release Notes` button to auto-populate the notes. Add anything else.
5. Tick the "This is a pre-release" checkbox. _Note: This doesn't functionally do anything right now, it's just a Github UI thing._
6. Click on the publish release button. **The [CI scripts](../.circleci) will automatically build and attach binaries.**
7. Download the binaries and test that they do what you expect. `curl -sL https://get.bacalhau.org/install.sh | bash`. You **must** update the ops deployments to make the new version work because of signature errors. See [troubleshooting below](#hints-tips-and-troubleshooting).
8. Edit the release and de-select the "This is a pre-release" checkbox
9. [Update the Bacalhau servers -- see below](#updating-the-bacalhau-networks).

## Updating the Bacalhau Networks

**TL;DR**: Allow the CI to deploy development and staging networks. Manually deploy the production network. [See the docs](../ops/README.md).

There are three environments: development, staging and production. For more information see the [ops documentation](../ops/README.md).

1. Do your infrastructure development (if any) in the development environment.
1. When ready, commit changes to Git for both the [development](../ops/terraform/development.tfvars) and [staging](../ops/terraform/staging.tfvars) `tfvars` files. For example, alter the `bacalhau_version` variable in the `development.tfvars` and `staging.tfvars` file.
1. Wait for the CI scripts to release the new infrastructure.
1. Once you are happy staging is working as intended, make changes to the [production](../ops/terraform/production.tfvars) `tfvars` file but DO NOT COMMIT yet.
1. [Manually apply the changes to the production environment.](../ops/README.md#deploying-bacalhau-mainnet) Please note that it takes a couple of minutes for the init scripts to install and start the Bacalhau servers. You can see what the server is doing with `gcloud compute ssh bacalhau-vm-$WORKSPACE-0 -- journalctl -f`.
1. Once the infrastructure is deployed, commit the changes to Git.

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
