# Bacalhau terraform

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

## gcloud config

It's important that when you are working on a particular bacalhau cluster - that your `gcloud` CLI is connected to the correct project and compute zone.

The `scripts/connect_project.sh` script is for this - it will connect to the gcloud project and zone mentioned in a `.tfvars` file

For example - `production.tfvars`:

```
gcp_project           = "bacalhau-production"
instance_count        = 3
region                = "us-east4"
zone                  = "us-east4-c"
```

If we run:

```bash
bash scripts/connect_project.sh production
```

It will do the following for us automatically:

```bash
gcloud config set project $(get_variable gcp_project)
gcloud config set compute/zone $(get_variable zone)
```

Make sure that when you start to work with a 

## terraform

Requires:
  * [terraform](https://www.terraform.io/downloads)

The `ops/terraform` directory contains the terraform configuration and all of the logic lives in `main.tf`.

```bash
cd ops/terraform
terraform init
terraform workspace list
```

Terraform state is managed using workspaces - there is a GCS bucket called `bacalhau-global-storage` the lives in the `bacalhau-cicd` project that keeps the tfstate for each workspace.

Combined with a `<workspace-name>.tfvars` variables file that controls which google project we deploy to - we can manage multiple bacalhau clusters into the same gcloud project.

# Deploying Bacalhau mainnet!

```bash
# make sure you are logged into the google user that has access to our gcloud projects
gcloud auth application-default login
# make sure gcloud is connected to the correct project and compute zone for our workspace
bash scripts/gcloud_connect.sh production
cd terraform
terraform init
# switch to the correct terraform workspace state file
terraform workspace select production
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
# make sure gcloud is connected to the correct project and compute zone for our workspace
bash scripts/gcloud_connect.sh $WORKSPACE
# create a new workspace state file for this cluster
terraform workspace new $WORKSPACE
# get the first node up and running
terraform apply \
  -var-file $WORKSPACE.tfvars \
  -var="bacalhau_node0_id=" \
  -var="bacalhau_connect=false" \
  -var="instance_count=1"
# now we need to get the libp2p id of the first node
gcloud compute ssh bacalhau-vm-$WORKSPACE-0 -- cat /tmp/bacalhau.log | grep "peer id is" | awk -F': ' '{print $2}'
# copy this id and paste it into the variables file
# edit variables
#   * bacalhau_node0_id = <id copied from SSH command above>
vi $WORKSPACE.tfvars
# now we re-apply the terraform command
terraform apply \
  -var-file $WORKSPACE.tfvars
```
