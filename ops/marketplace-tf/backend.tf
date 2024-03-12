terraform {
  backend "gcs" {
    bucket = "bacalhau-infra-state"
    prefix = "terraform"
  }
}
