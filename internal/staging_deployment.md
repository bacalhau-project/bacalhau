# Deploy to Staging Environment

Staging is a separate environment and network of nodes with a dedicated canary that can be helpful to test features and changes before deploying to production. Jobs submitted to staging are not picked up by production nodes, and vice versa.

***It is highly recommended to test critical and potentially perf impacting changes in staging for few hours or days before deploying to production, or even merging into main***



## Instructions
We are using Terraform to update our infrastructure. More information about setting up your Terraform workspaces can be found [here](../ops/README.md).

```bash
cd ops/terraform

# make sure gcloud is connected to the correct project and compute zone for our workspace
bash scripts/connect_workspace.sh staging

# apply the latest variables
terraform plan -var-file staging.tfvars
terraform apply -var-file staging.tfvars

# (optional) if you want to deploy with updated secrets
terraform plan -var-file staging.tfvars -var-file secrets.tfvars
terraform apply -var-file staging.tfvars -var-file secrets.tfvars

# (optional) tail bacalhau logs
# Please note that it takes a few minutes for the init scripts to install and start the Bacalhau servers and for the logs to appear
gcloud compute ssh bacalhau-vm-staging-0 -- journalctl -f bacalhau

# (optional) tail init scripts progress
# You can check the progress of init scripts. Note that this file is deleted after Bacalhau server installation is successful
gcloud compute ssh bacalhau-vm-staging-0 --  tail -f /tmp/bacalhau.log
```

## Deploy from a different branch
Staging is configured to deploy from the tip of `main`, so you don't need to create a pre-release to test on staging. You can change [staging.tfvars](../ops/terraform/staging.tfvars) to deploy from a different branch. You don't need to commit and push changing the branch, but you might want to communicate that over our slack channel to make sure no one else deploys and replaces your changes.

## Monitor staging
Staging has a dedicate canary and you can monitor the health and performance of your changes using the [Canary Dashboard](https://cloudwatch.amazonaws.com/dashboard.html?dashboard=BacalhauCanaryStaging&context=eyJSIjoidXMtZWFzdC0xIiwiRCI6ImN3LWRiLTI4NDMwNTcxNzgzNSIsIlUiOiJ1cy1lYXN0LTFfUTlPMEVrM3llIiwiQyI6IjExc3NlYW1tZmVmaGdtYTFzMDk1c29jaDltIiwiSSI6InVzLWVhc3QtMTo2Nzk5ODFmZC03ZjZlLTRmYjItOTY3Ny1iNjYxMDA2NjBlZjgiLCJNIjoiUHVibGljIn0%3D)
