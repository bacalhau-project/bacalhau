provider "google" {
  project = var.gcp_project_id
  region  = var.gcp_region
  zone = var.gcp_zone
}

module "gcp_network" {
  source      = "./modules/gcp/network"
  region      = var.gcp_region
  subnet_cidr = "10.0.0.0/16" // Example CIDR, adjust as needed
}

module "requester_instance" {
  source = "./modules/gcp/compute_instances/requester"
  cloud_init_content = ""

  aws_credentials = local.aws_credentials
  build_config = local.build_config
  token_config = local.token_config
  gcp_config = local.gcp_config

  disk_config = {
    boot_size = var.bacalhau_boot_disk_size
    repo_size = var.bacalhau_repo_disk_size
  }

  tls_config = {
    bacalhau_tls_crt = tls_self_signed_cert.tlscert.cert_pem
    bacalhau_tls_sk = tls_private_key.privkey.private_key_pem
  }

  requester_static_ip = module.gcp_network.requester_ip
  requester_instance_type = var.requester_machine_type

  bacalhau_accept_networked_jobs = var.bacalhau_accept_networked_jobs
  bacalhau_otel_collector_endpoint = var.bacalhau_otel_collector_endpoint

}

module "compute_instance" {
  source = "./modules/gcp/compute_instances/compute"
  cloud_init_content = ""

  aws_credentials = local.aws_credentials
  build_config = local.build_config
  token_config = local.token_config
  gcp_config = local.gcp_config

  disk_config = {
    boot_size = var.bacalhau_boot_disk_size
    repo_size = var.bacalhau_repo_disk_size
    local_size = var.bacalhau_local_disk_size
  }

  // This creates an implicit dependency, meaning Terraform will create the requester_instance before the compute_instance.
  requester_ip = module.requester_instance.requester_private_ips[0]
  compute_instance_count = var.compute_count
  compute_instance_type = var.compute_machine_type

  bacalhau_accept_networked_jobs = var.bacalhau_accept_networked_jobs
  bacalhau_otel_collector_endpoint = var.bacalhau_otel_collector_endpoint
  bacalhau_tls_crt = tls_self_signed_cert.tlscert.cert_pem
}

locals {
  token_config = {
    requester_api_token = var.bacalhau_requester_api_token != "" ? var.bacalhau_requester_api_token : random_string.bacalhau_requester_api_token.result
    compute_api_token = var.bacalhau_compute_api_token != "" ? var.bacalhau_compute_api_token : random_string.bacalhau_compute_api_token.result
  }
  build_config = {
    install_version = var.bacalhau_install_version
    install_branch = var.bacalhau_install_branch
    install_commit = var.bacalhau_install_commit
  }
  aws_credentials = {
    access_key_id = var.aws_access_key_id
    secret_access_key = var.aws_secret_access_key
  }
  gcp_config = {
    network = module.gcp_network.vpc_network_name
    subnetwork = module.gcp_network.subnetwork_name
    zone = var.gcp_zone
    boot_image = var.gcp_boot_image
  }
}

resource "random_string" "bacalhau_requester_api_token" {
  length  = 32
  special = false
  # Only generate a new random string if no bacalhau_client_access_token is provided
  keepers = {
    token = var.bacalhau_requester_api_token == "" ? "generate" : "provided"
  }
}

resource "random_string" "bacalhau_compute_api_token" {
  length  = 32
  special = false
  # Only generate a new random string if no bacalhau_client_access_token is provided
  keepers = {
    token = var.bacalhau_compute_api_token == "" ? "generate" : "provided"
  }
}


resource "tls_private_key" "privkey" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "tls_self_signed_cert" "tlscert" {
  private_key_pem = tls_private_key.privkey.private_key_pem

  subject {
    common_name  = module.gcp_network.requester_ip
  }

  validity_period_hours = 8760 // 365 days

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "server_auth",
  ]

  ip_addresses = [module.gcp_network.requester_ip]

  is_ca_certificate = false
}


