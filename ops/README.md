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