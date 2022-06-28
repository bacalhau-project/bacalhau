variable "bacalhau_version" {
  type = string
}
variable "ipfs_version" {
  type = string
}
variable "gcp_project" {
  type = string
}
variable "machine_type" {
  type = string
}
variable "instance_count" {
  type = string
}
variable "volume_size_gb" {
  type = number
}
variable "boot_disk_size_gb" {
  type = number
  default = 10
}
variable "restore_from_backup" {
  type    = string
  default = ""
}
variable "region" {
  type = string
}
variable "zone" {
  type = string
}

variable "ingress_cidrs" {
  type    = set(string)
  default = []
}
variable "ssh_access_cidrs" {
  type    = set(string)
  default = []
}

variable "rollout_phase" {
  type = string
}
