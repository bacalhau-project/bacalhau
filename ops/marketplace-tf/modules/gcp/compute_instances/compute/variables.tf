variable "aws_credentials" {
  description = "AWS credentials"
  type = object({
    access_key_id     = string
    secret_access_key = string
  })
}

variable "build_config" {
  description = "Configuration for building specific versions of bacalhau"
  type = object({
    install_version = string
    install_branch = string
    install_commit = string
  })
}

variable "token_config" {
  description = "Configuration for setting up auth tokens"
  type = object({
    requester_api_token = string
    compute_api_token = string
  })
}

variable "bacalhau_tls_crt" {
  description = "Certificate for TLS"
  type = string
}

variable "gcp_config" {
  description = "Configuration specific to GCP including networking and boot image"
  type = object({
    network = string
    subnetwork = string
    zone = string
    boot_image = string
  })
}

variable "disk_config" {
  description = "Configuration related to local storage disk, repo disk, and boot disk"
  type = object({
    boot_size = number
    repo_size= number
    local_size = number
  })
}

variable "compute_instance_count" {
  description = "Number of compute instances"
  type        = number
}

variable "compute_instance_type" {
  description = "The instance type for the compute"
  type        = string
}

variable "requester_ip" {
  description = "Private IP of the requester instance"
  type        = string
}


variable "cloud_init_content" {
  description = "Content of the cloud-init script"
  type        = string
}

variable "bacalhau_accept_networked_jobs" {
  description = "When true bacalhau will accept jobs requiring networking. Otherwise they will be rejected."
  type = bool
  default = false
}

variable "bacalhau_otel_collector_endpoint" {
  description = "The opentelemetry collector endpoint to send metrics to"
  type = string
}