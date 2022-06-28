# Deploying Bacalhau mainnet!

Requires:
* terraform
* gcloud credentials set up to access `Bacalhau - Production` project

```
cd terraform
```

```
gcloud auth application-default login
```

```
terraform init
```

```
terraform workspace select production
terraform apply -var-file production.tfvars
```

TODO:
* [x] State file in GCS
* [x] Increase disk quota
* [x] Actually use the attached disks for ipfs
* [x] Write bacalhau keypair to attached disk
