variable "gcp_project" {
    type = string
}
variable "machine_type" {
    type = string
    default = "e2-standard-4"
}
variable "instance_count" {
    type = string
    default = "2"
}
variable "volume_size_gb" {
    type = number
    default = "10"
}
variable "restore_from_backup" {
    type = string
    default = ""
}
variable "region" {
    type = string
    default = "europe-north1"
}
variable "zone" {
    type = string
    default = "europe-north1-c"
}

variable "ingress_cidrs" {
    type = set(string)
    default = []
}
variable "ssh_access_cidrs" {
    type = set(string)
    default = []
}