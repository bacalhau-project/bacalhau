terraform {
  backend "gcs" {
    bucket = "bacalhau-otel-collector-infra-state"
    prefix = "terraform"
  }
}
