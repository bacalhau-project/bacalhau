variable "instance_count" {
    type = string
}
variable "zone" {
    type = string
}
variable "volume_size" {
    type = string
}
variable "restore_from_backup" {
    type = string
    default = ""
}
variable "region" {
    type = string
}
variable "ingress_cidrs" {
    type = list(string)
}
variable "ssh_access_cidrs" {
    type = list(string)
}