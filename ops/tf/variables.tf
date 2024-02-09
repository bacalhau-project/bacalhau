variable "gcp_project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "gcp_region" {
  description = "GCP Region"
  type        = string
}

variable "gcp_zone" {
  description = "GCP Zone"
  type        = string
}

variable "gcp_boot_image" {
  description = "Boot image for GCP instances"
  type        = string
}

variable "requester_machine_type" {
  description = "Machine type for requester instances"
  type        = string
}

variable "compute_machine_type" {
  description = "Machine type for compute instances"
  type        = string
}

variable "compute_count" {
  description = "Number of compute instances"
  type        = number
}

variable "aws_access_key_id" {
  description = "AWS access key id used to authenticate s3 compatible storage"
  type = string
}

variable "aws_secret_access_key" {
  description = "AWS secret access key used to authenticate s3 compatible storage"
  type = string
}

variable "bacalhau_accept_networked_jobs" {
  description = "When true bacalhau will accept jobs requiring networking. Otherwise they will be rejected."
  type = bool
  default = false
}

variable "bacalhau_boot_disk_size" {
  description = "The size of the boot disk in GB."
  type        = number
  default = 50
}

variable "bacalhau_repo_disk_size" {
  description = "The size of the disk in GB bacalhau will to store its repo"
  type        = number
  default = 50
}

variable "bacalhau_local_disk_size" {
  description = "The size of the disk in GB bacalhau will to store local data"
  type        = number
  default = 50
}

variable "bacalhau_otel_collector_endpoint" {
  description = "The opentelemetry collector endpoint to send metrics to"
  type = string
}


variable "bacalhau_install_version" {
  description = "The version of bacalhau to install. If empty default to https://get.bacalhau.org/install.sh"
  type = string
  default = ""
}

variable "bacalhau_install_branch" {
  description = "The branch of bacalhau to install. If empty default to https://get.bacalhau.org/install.sh"
  type = string
  default = ""
}

variable "bacalhau_install_commit" {
  description = "The commit sha of bacalhau to install. If empty default to https://get.bacalhau.org/install.sh"
  type = string
  default = ""
}

variable "bacalhau_requester_api_token" {
  description = <<EOF
  (Optional) Provide a token used for clients to connect to the requester. If blank one will be generated as part of
  the deployment and displayed in the output.
  EOF
  type = string
  default = ""
}

variable "bacalhau_compute_api_token" {
  description = <<EOF
  (Optional) Provide a token used for compute nodes to connect to the requester. If blank one will be generated as
part of the deployment and displayed in the output.
  EOF
  type = string
  default = ""
}