# Bacalhau terraform

Requires:
  * [terraform](https://www.terraform.io/downloads)
  * [gcloud](https://cloud.google.com/sdk/docs/install)

## gcloud projects

Install [gcloud](https://cloud.google.com/sdk/docs/install) and login:

```bash
gcloud auth application-default login
```

Projects you will need access to:

 * [bacalhau-cicd](https://console.cloud.google.com/welcome?project=bacalhau-cicd)
   * this is where the GCS bucket with terraform state lives
 * [bacalhau-development](https://console.cloud.google.com/welcome?project=bacalhau-development)
   * commits to main are deployed here
   * scale tests are run here
   * short lived clusters
 * [bacalhau-staging](https://console.cloud.google.com/welcome?project=bacalhau-staging)
   * long lived cluster
 * [bacalhau-production](https://console.cloud.google.com/welcome?project=bacalhau-production)
   * long lived cluster

## terraform workspaces

The `ops/terraform` directory contains the terraform configuration and all of the logic lives in `main.tf`.

```bash
cd ops/terraform
gcloud auth application-default login
terraform init
terraform workspace list
```

Terraform state is managed using workspaces - there is a GCS bucket called `bacalhau-global-storage` the lives in the `bacalhau-cicd` project that keeps the tfstate for each workspace.

Combined with a `<workspace-name>.tfvars` variables file that controls which google project we deploy to - we can manage multiple bacalhau clusters into the same gcloud project.

## bacalhau workspaces

A bacalhau workspace is a combination of:

 * `<workspace-name>.tfvars` - the variables controlling the versions, cluster size, gcloud project, gcloud region
 * `<workspace-name>-secrets.tfvars` - the sensitive API keys that are required (not checked in to source control)
 * a terraform workspace named `<workspace-name>` which points at the state file managed in the GCS bucket

The `bash scripts/connect_workspace.sh <workspace-name>` will connect to the correct gcloud project and zone named in `<workspace-name>.tfvars` and run `terraform workspace select <workspace-name>` so you can begin to work with that cluster.

**IMPORTANT:** always run `bash scripts/connect_workspace.sh` before running `terraform` commands for a given project.

```bash
bash scripts/connect_workspace.sh production
terraform plan -var-file production.tfvars -var-file production-secrets.tfvars
```

# Deploying Bacalhau mainnet

The normal operation is to edit `production.tfvars`, make sure the `bacalhau_version` variable points to the version you'd like to deploy.
Optionally, ensure you have appropriate values in `production-secrets.tfvars` (see `secrets.tfvars.example` for a guide) and then:

```bash
# make sure gcloud is connected to the correct project and compute zone for our workspace
bash scripts/connect_workspace.sh production
# apply the latest varibales
terraform plan -var-file production.tfvars -var-file production-secrets.tfvars
terraform apply -var-file production.tfvars -var-file production-secrets.tfvars
```

> :warning: Due to some limitations in how GCP provision gpus (inquiry @simonwo for more details :smile:) the disk of one of the gpu machines [has to be restored from a hand-picked snapshot](https://github.com/bacalhau-project/bacalhau/blob/587415f600ba8b1b4a117799d1a14907430b893c/ops/terraform/main.tf#L198). This is a temporary solution.

# Stand up a new long lived cluster

To start a new long lived cluster - we need to first standup the first node and get it's libp2p id and then re-apply the cluster

```bash
# make sure you are logged into the google user that has access to our gcloud projects
gcloud auth application-default login
# the name of the cluster (and workspace)
export WORKSPACE=apples
cp staging.tfvars $WORKSPACE.tfvars
# edit variables
#   * gcp_project = bacalhau-development
#   * region = XXX
#   * zone = XXX
vi $WORKSPACE.tfvars
# create a new workspace state file for this cluster
terraform workspace new $WORKSPACE
# make sure gcloud is connected to the correct project and compute zone for our workspace
bash scripts/connect_workspace.sh $WORKSPACE
# get the first node up and running
terraform apply \
  -var-file $WORKSPACE.tfvars \
  -var-file $WORKSPACE-secrets.tfvars \
  -var="bacalhau_connect_node0=" \
  -var="instance_count=1"
# wait a bit of time so the bacalhau server is up and running
sleep 10
gcloud compute ssh bacalhau-vm-$WORKSPACE-0 -- sudo systemctl status bacalhau
# now we need to get the libp2p id of the first node
gcloud compute ssh bacalhau-vm-$WORKSPACE-0 -- journalctl -u bacalhau | grep "peer id is" | awk -F': ' '{print $2}'
# copy this id and paste it into the variables file
# edit variables
#   * bacalhau_connect_node0 = <id copied from SSH command above>
vi $WORKSPACE.tfvars
# now we re-apply the terraform command
terraform apply \
  -var-file $WORKSPACE.tfvars \
  -var-file $WORKSPACE-secrets.tfvars
```

# Deleting long lived cluster

There is `prevent_destroy = true` on long lived clusters.

This is controlled by the `protect_resources = true` variable.

The only way to delete a long lived cluster (because you've thought hard about it and have decided it is actually what you want to do) is to edit the `main.tf` file and set `prevent_destroy = false` on the ip address and the disk before doing:

```bash
terraform destroy -var-file $WORKSPACE.tfvars
```

**IMPORTANT:** remember to reset `prevent_destroy = true` in `main.tf` (please don't commit it with `prevent_destroy = false`)

Once you have deleted a cluster - don't forget to:

```bash
terraform workspace delete $WORKSPACE
rm -f $WORKSPACE.tfvars
```

# Stand up a new short lived cluster

This is for scale tests or short lived tests on a live network.

We set `bacalhau_unsafe_cluster=true` so nodes automatically connect to each other (it uploads a fixed, unsafe private key from this repo so we know the libp2p id of node0)

We set `protect_resources=false` so we can easily delete the cluster when we are done.

```bash
export WORKSPACE=oranges
cp shortlived_example.tfvars $WORKSPACE.tfvars
# edit variables
#   * gcp_project = bacalhau-development
#   * region = XXX
#   * zone = XXX
vi $WORKSPACE.tfvars
# create a new workspace state file for this cluster
terraform workspace new $WORKSPACE
# make sure gcloud is connected to the correct project and compute zone for our workspace
bash scripts/connect_workspace.sh $WORKSPACE
# get the first node up and running
terraform apply \
  -var-file $WORKSPACE.tfvars
sleep 10
gcloud compute ssh bacalhau-vm-$WORKSPACE-0 -- sudo systemctl status bacalhau
```

# Deleting short lived cluster

```bash
export WORKSPACE=oranges
bash scripts/connect_workspace.sh $WORKSPACE
terraform destroy \
  -var-file $WORKSPACE.tfvars
terraform workspace select development
terraform workspace delete $WORKSPACE
rm $WORKSPACE.tfvars
```

# Debugging startup issues

To see the logs from a nodes startup script:

```bash
export WORKSPACE=apples
bash scripts/connect_workspace.sh $WORKSPACE
gcloud compute ssh bacalhau-vm-$WORKSPACE-0 -- sudo journalctl -u google-startup-scripts.service
```

# Backwards compatible naming

In some resources, the `name` property of a resource is calculated like this:

```
name  = terraform.workspace == "production" ?
  "bacalhau-ipv4-address-${count.index}" :
  "bacalhau-ipv4-address-${terraform.workspace}-${count.index}"
```

This is a backwards compatible mode to preserve the production disks and ip addresses by avoiding renaming them.

# Protected resources

The disks and ip addresses are in one of two modes:

 * `protected`
 * `unprotected`

To control which type is used - you set the `protect_resources` variable to true when creating a cluster.

# Auto subnets

With long lived clusters - we use the `auto_subnets = true` setting which means there will be a bunch of subnetworks auto created for the deployment network.

For short lived clusters - we set this to false and create a single manual sub network.

This is so we don't use up all of our network quota making subnets that we don't actually use.

# Uploading CIDs

Sometimes it's useful to upload content directly to nodes in a terraform managed cluster.

There is a script to help do that:

```bash
bash scripts/upload_cid.sh production ~/path/to/local/content
```

# Troubleshoot production

To inspect the aggregated logs in Grafana Cloud access [this dashboard](https://protocollabs.grafana.net/goto/lKmGkWT4z?orgId=1) (requires credentials!).


Alternatively, you need to ssh into the hosts in the [bacalhau-production](https://console.cloud.google.com/welcome?project=bacalhau-production) project. Inspect the logs with `journalctl -u bacalhau -f`.
