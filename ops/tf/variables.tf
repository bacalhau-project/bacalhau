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

variable "gcp_boot_image_requester" {
  description = "Boot image for GCP requester instances"
  type        = string
}

variable "gcp_boot_image_compute" {
  description = "Boot image for GCP Compute instances"
  type        = string
}

variable "accelerator" {
  description = "Accelerator for GCP Compute instances"
  type        = string
}
variable "accelerator_count" {
  description = "Accelerator Count for GCP Compute instances, 0 Implies no accelerator"
  type        = number
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

variable "bacalhau_repo_disk_size" {
  description = "The size of the disk in GB bacalhau will to store its repo"
  type        = number
  default = 50
}
