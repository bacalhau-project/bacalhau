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
