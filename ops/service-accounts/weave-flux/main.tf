provider "google" {
  project = var.gcp_project
  region  = var.region
  zone    = var.zone
}

terraform {
  backend "gcs" {
    # this bucket lives in the bacalhau-cicd google project
    # https://console.cloud.google.com/storage/browser/bacalhau-global-storage;tab=objects?project=bacalhau-cicd
    bucket = "bacalhau-global-storage"
    prefix = "terraform/state"
  }
}

resource "google_service_account" "sa" {
  account_id   = var.service_account_name
  display_name = "Service Account For Weave Flux"
}

resource "google_service_account_key" "sak" {
  service_account_id = google_service_account.sa.name
}

resource "google_project_iam_binding" "compute_role" {
  project = var.gcp_project
  role    = "roles/compute.admin"

  members = [
    "serviceAccount:${google_service_account.sa.email}",
  ]
}

resource "google_project_iam_binding" "terraform_state_role" {
  project = "bacalhau-cicd"
  role    = "roles/storage.admin"

  members = [
    "serviceAccount:${google_service_account.sa.email}",
  ]
}

resource "local_file" "key_file" {
  content  = base64decode(google_service_account_key.sak.private_key)
  filename = "${path.module}/${var.service_account_name}-${terraform.workspace}.json"
}
