# Weave Service Accounts

Weave will need a service account in each account that has an ops cluster.

## Applying

```
cd ops/service-accounts/weave-flux
export WORKSPACE=development
export VARIABLES_FILE=../../terraform/$WORKSPACE.tfvars
bash ../../terraform/scripts/connect_workspace.sh $WORKSPACE
terraform workspace select $WORKSPACE-weave-flux
terraform apply -var-file ../../terraform/$WORKSPACE.tfvars -compact-warnings
```

Ignore the warnings about undeclared variables.

It will output a json file called `service-account-weave-flux-development-weave-flux.json` that contains the service account key. Send this to whoever needs it securely.

Then repeat for `staging` and `production` when you're ready.