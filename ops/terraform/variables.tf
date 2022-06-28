variable "bacalhau_version" {
  type = string
}
variable "bacalhau_port" {
  type = string
}
# this means that we are using long lived keys in a production type network
# if this is not defined - then we will use the unsafe private key
# for the first node in the cluster and so we can know the id to connect
# other nodes to
variable "bacalhau_node0_id" {
  type = string
}
# should we automatically connect nodes or should we just stand up nodes
# on their own - this is used to initially stand up a cluster so we can
# get the ID of the first node to then pass in as a variable
# (alongside then turning on bacalhau_connect to connect other nodes to the fisrt one)
variable "bacalhau_connect" {
  type = bool
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
# IMPORTANT - in production you really do want this to be true
variable "keep_disks" {
  type = bool
}
variable "volume_size_gb" {
  type = number
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
