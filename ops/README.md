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
terraform apply -var-file production.tfvars
```

TODO:
* State file in GCS
* Increase disk quota
* Actually use the attached disks for ipfs
* Write bacalhau keypair to attached disk