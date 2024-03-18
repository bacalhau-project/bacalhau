variable "region" {
  description = "The region to host the network in"
  type        = string
}

variable "subnet_cidr" {
  description = "The CIDR block for the subnet"
  type        = string
}

variable "auto_subnets" {
  description = "When true GCP will automatically create subnetworks"
  type    = bool
  default = true
}

//
// Egress
//
variable "egress_tcp_ports" {
  description = "List of TCP ports for egress rules"
  type        = list(string)
  default     = [
    // Grafana
    "443"
  ]
}

variable "egress_source_ranges" {
  description = "Source ranges for egress rules"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

//
// Ingress
//
variable "ingress_tcp_ports" {
  description = "List of TCP ports for ingress rules"
  type        = list(string)
  default     = [
    // SSH
    "22",
    // OpenTelemetry collector
    "4318"
  ]
}

variable "ingress_source_ranges" {
  description = "Source ranges for ingress rules"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}
