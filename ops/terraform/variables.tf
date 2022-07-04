variable "bacalhau_version" {
  type = string
}
variable "bacalhau_port" {
  type = string
}
# used to quickly provision a connected cluster using the unsafe private key
# IMPORTANT - only use this for test clusters or stress test clusters
# it will result in node0 having an unsafe private key
variable "bacalhau_unsafe_cluster" {
  type = bool
  default = false
}
# connect to a known node0 id
# this is used for long lived clusters that have already been bootstrapped
# and the node0 id is derived from a persisted known private key
variable "bacalhau_connect_node0" {
  type = string
  default = ""
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
// should we add delete protection to public ip addresses and disks?
variable "protect_resources" {
  type    = bool
  default = true
}
// should we automatically make subnets (for long lived clusters)
// set to false if this is a short lived cluster
variable "auto_subnets" {
  type    = bool
  default = false
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