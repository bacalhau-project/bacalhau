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
    "4001", // ipfs
    "1235", // bacalhau
    "4318", // otel
  ]
}

variable "egress_udp_ports" {
  description = "List of UDP ports for egress rules"
  type        = list(string)
  default     = [
    // HTTP(s)
    "80",   // webui & otel (TODO switch to HTTPS)
    "443",
    // ipfs daemon
    "4001",
    // NATs
    "6222",
    "4222"
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
    // HTTP(s)
    "80",   // webui & otel (TODO switch to HTTPS)
    "443",
    // ipfs daemon
    "4001",
    // bacalhau
    "1235",
    // API
    "1234",
    // Metrics
    "13133",
    "55679",
    // Health
    "44443",
    "44444",
    // NATs
    "6222",
    "4222"
  ]
}

variable "ingress_udp_ports" {
  description = "List of UDP ports for ingress rules"
  type        = list(string)
  default     = [
    // ipfs daemon
    "4001",
    // bacalhau
    "1235",
  ]
}


variable "ingress_source_ranges" {
  description = "Source ranges for ingress rules"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}
