variable "requester_instance_type" {
  description = "The instance type for the requester"
  type        = string
}

variable "requester_static_ip" {
  description = "The static IP address for the requester instance"
  type        = string
}

variable "zone" {
  description = "The zone in which to provision instances"
  type        = string
}

variable "boot_image" {
  description = "The boot image for the instances"
  type        = string
  default     = "projects/ubuntu-os-cloud/global/images/family/ubuntu-2304-amd64"
}

variable "boot_size" {
  description = "The size of the boot disk"
  type        = number
  default     = 50
}

variable "cloud_init_content" {
  description = "Content of the cloud-init script"
  type        = string
}

variable "network" {
  description = "The VPC network to attach to the instances"
  type        = string
}

variable "subnetwork" {
  description = "The subnetwork to attach to the instances"
  type        = string
}

variable "install_bacalhau_argument" {
  description = "Argument to pass to the install bacalhau script"
  type        = string
  // Usage: install-bacalhau [release <version> | branch <branch-name>]
  default     = ""
}


