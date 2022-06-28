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
terraform apply -var-file production.tfvars
```

TODO:
* [ ] State file in GCS
* [ ] Increase disk quota
* [x] Actually use the attached disks for ipfs
* [x] Write bacalhau keypair to attached disk


# Run a scale test

```
gcloud auth login
gcloud config set project bacalhau-development
gcloud config set compute/zone europe-north1-c
```