variable "gcp_project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "gcp_region" {
  description = "GCP Region"
  type        = string
}

variable "gcp_zone" {
  description = "GCP Zone"
  type        = string
}

variable "gcp_boot_image" {
  description = "Boot image for GCP instances"
  type        = string
  default = "projects/ubuntu-os-cloud/global/images/family/ubuntu-2304-amd64"
}

variable "otel_collector_machine_type" {
  description = "Machine type for collector instances"
  type        = string
}

variable "grafana_prometheus_username" {
  description = "username for hosted grafana prometheus"
  type = string
}

variable "grafana_prometheus_password" {
  description = "password for hosted grafana prometheus"
  type = string
}
