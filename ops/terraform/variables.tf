variable "gcp_project" {
    type = string
}
variable "machine_type" {
    type = string
}
variable "instance_count" {
    type = string
}
variable "zone" {
    type = string
}
variable "volume_size_gb" {
    type = number
}
variable "restore_from_backup" {
    type = string
    default = ""
}
variable "region" {
    type = string
}
variable "ingress_cidrs" {
    type = set(string)
    default = []
}
variable "ssh_access_cidrs" {
    type = set(string)
    default = []
}