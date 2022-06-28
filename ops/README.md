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
 * a terraform workspace named `<workspace-name>` which points at the state file managed in the GCS bucket

The `bash scripts/connect_workspace.sh <workspace-name>` will connect to the correct gcloud project and zone named in `<workspace-name>.tfvars` and run `terraform workspace select <workspace-name>` so you can begin to work with that cluster.

**IMPORTANT:** always run `bash scripts/connect_workspace.sh` before running `terraform` commands for a given project.

```bash
bash scripts/connect_workspace.sh production
terraform plan -var-file production.tfvars
```

# Deploying Bacalhau mainnet!

The normal operation is to edit `production.tfvars` and then:

```bash
# make sure gcloud is connected to the correct project and compute zone for our workspace
bash scripts/connect_workspace.sh production
# apply the latest varibales
terraform apply -var-file production.tfvars
```

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
  -var="bacalhau_connect_node0=" \
  -var="instance_count=1"
# wait a bit of time so the bacalhau server is up and running
sleep 10
gcloud compute ssh bacalhau-vm-$WORKSPACE-0 -- sudo systemctl status bacalhau-daemon
# now we need to get the libp2p id of the first node
gcloud compute ssh bacalhau-vm-$WORKSPACE-0 -- cat /tmp/bacalhau.log | grep "peer id is" | awk -F': ' '{print $2}'
# copy this id and paste it into the variables file
# edit variables
#   * bacalhau_connect_node0 = <id copied from SSH command above>
vi $WORKSPACE.tfvars
# now we re-apply the terraform command
terraform apply \
  -var-file $WORKSPACE.tfvars
```

# Stand up a new short lived cluster

This is for scale tests or short lived tests on a live network.

```bash
# make sure you are logged into the google user that has access to our gcloud projects
gcloud auth application-default login
# the name of the cluster (and workspace)
export WORKSPACE=oranges
cp staging.tfvars $WORKSPACE.tfvars
# edit variables
#   * gcp_project = bacalhau-development
#   * region = XXX
#   * zone = XXX
#   * bacalhau_unsafe_cluster = true
# IMPORTANT - make sure you set bacalhau_unsafe_cluster = true
vi $WORKSPACE.tfvars
# create a new workspace state file for this cluster
terraform workspace new $WORKSPACE
# make sure gcloud is connected to the correct project and compute zone for our workspace
bash scripts/connect_workspace.sh $WORKSPACE
# get the first node up and running
terraform apply \
  -var-file $WORKSPACE.tfvars
```

# Deleting clusters

For the moment - we have `prevent_destroy = true` on both the disks and ip addresses.

This will prevent a `terraform destroy -var-file $WORKSPACE.tfvars` command from working.

Until we have a better solution - edit `main.tf` and update to `prevent_destroy = false`

**IMPORTANT:** remember to set `prevent_destroy = false` and don't commit `main.tf` with `prevent_destroy = true`

TODO: find a way to to this better and avoid commits that set `prevent_destroy = false`

Once you have deleted a cluster - don't forget to:

```bash
terraform workspace delete $WORKSPACE
rm -f $WORKSPACE.tfvars
```