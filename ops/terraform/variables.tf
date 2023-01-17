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
  type    = bool
  default = false
}
# these are used for long lived clusters that have already been bootstrapped
# and the node0, node1 and node2 ids are derived from a persisted known private key
variable "bacalhau_node_id_0" {
  type    = string
  default = ""
}
variable "bacalhau_node_id_1" {
  type    = string
  default = ""
}
variable "bacalhau_node_id_2" {
  type    = string
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
  type    = number
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

// secrets - if these are set then they will get injected into a secrets file
// on the node's persistent data disk. This is useful for initialising stuff
// like API keys that shouldn't go in the public repo.
variable "honeycomb_api_key" {
  type      = string
  default   = ""
  sensitive = true
}

// Out of a total of var.instance_count machines, how many do you want to be GPU machines?
// I chose this, rather than making a new pool of machines, to maintain configuration parity
variable "num_gpu_machines" {
  type    = number
  default = 0
}

// Number of GPUs attached to each machine
variable "num_gpus_per_machine" {
  type    = number
  default = 1
}

// The sku of the GPU
variable "gpu_type" {
  type    = string
  default = "nvidia-tesla-t4"
}

// The machine type to attach the GPU to. Unfortunately not all machines support attaching GPUs. I suggest using the UI to figure this out.
variable "gpu_machine_type" {
  type    = string
  default = "n1-standard-4"
}

// Version number, omit the 'v' prefix
variable "prometheus_version" {
  type    = string
  default = ""
}

// Grafana: you can find the /api/prom/push URL, username, and password for your metrics
// endpoint by clicking on Details in the Prometheus card of the Cloud Portal
// https://grafana.com/docs/grafana-cloud/fundamentals/cloud-portal/
// Note: this is not an account-wide API key, but rather a key for Prometheus
variable "grafana_cloud_api_key" {
  type      = string
  default   = ""
  sensitive = true
}

variable "grafana_cloud_api_user" {
  type    = string
  default = ""
}

// Remote Write Endpoint 
// e.g. https://prometheus-prod-01-eu-west-0.grafana.net/api/prom/push
variable "grafana_cloud_api_endpoint" {
  type    = string
  default = ""
}

variable "grafana_cloud_loki_user" {
  type    = string
  default = ""
}

variable "grafana_cloud_loki_endpoint" {
  type    = string
  default = ""
}

variable "loki_version" {
  type    = string
  default = ""
}

variable "estuary_api_key" {
  type      = string
  default   = ""
  sensitive = true
}

variable "internal_ip_addresses" {
  type    = list(string)
  default = []
}
