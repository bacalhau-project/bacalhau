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
}

variable "bacalhau_local_disk_size" {
  description = "The size of the disk in GB bacalhau will to store local data"
  type        = number
}

variable "bacalhau_otel_collector_endpoint" {
  description = "The opentelemetry collector endpoint to send metrics to"
  type = string
}

variable "bacalhau_auth_token" {
  description = "Auth token for bacalhau api"
  type = string
}

variable "bacalhau_install_version" {
  description = "The version or branch of bacalhau to install. If empty https://get.bacalhau.org/install.sh will be used to install"
  type = string
  default = ""
}

variable "bacalhau_install_branch" {
  description = "The branch of bacalhau to install. If empty default to https://get.bacalhau.org/install.sh"
  type = string
  default = ""
}
