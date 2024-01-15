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

variable "install_bacalhau_argument" {
  description = "Argument to pass to the install bacalhau script"
  type        = string
  # Usage: install-bacalhau [release <version> | branch <branch-name>]
  default     = ""
}
