variable "compute_instance_count" {
  description = "Number of compute instances"
  type        = number
}

variable "compute_instance_type" {
  description = "The instance type for the compute"
  type        = string
}

variable "requester_ip" {
  description = "Private IP of the requester instance"
  type        = string
}

variable "zone" {
  description = "The zone in which to provision instances"
  type        = string
}

variable "boot_size" {
  description = "The size of the boot disk"
  type        = number
  default     = 50
}

variable "boot_image" {
  description = "The boot image for the instances"
  type        = string
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

