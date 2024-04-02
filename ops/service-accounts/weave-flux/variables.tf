variable "gcp_project" {
  type = string
}
variable "region" {
  type = string
}
variable "zone" {
  type = string
}
variable "service_account_name" {
  type    = string
  default = "service-account-weave-flux"
}
