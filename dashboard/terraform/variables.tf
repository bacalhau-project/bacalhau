variable "api_image" {
  type = string
}
variable "frontend_image" {
  type = string
}
variable "gcp_project" {
  type = string
}
variable "region" {
  type = string
}
variable "zone" {
  type = string
}
variable "machine_type" {
  type = string
}
variable "boot_disk_size_gb" {
  type    = number
  default = 100
}
variable "volume_size_gb" {
  type = number
  default = 100
}
variable "ingress_cidrs" {
  type    = set(string)
  default = []
}
variable "ssh_access_cidrs" {
  type    = set(string)
  default = []
}