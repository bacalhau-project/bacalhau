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

  network             = module.gcp_network.vpc_network_name
  subnetwork          = module.gcp_network.subnetwork_name
  requester_static_ip = module.gcp_network.requester_ip
  zone                = var.gcp_zone

  cloud_init_content = ""
  requester_instance_type = var.requester_machine_type
}

module "compute_instance" {
  source = "./modules/gcp/compute_instances/compute"

  network                 = module.gcp_network.vpc_network_name
  subnetwork              = module.gcp_network.subnetwork_name
  zone = var.gcp_zone

  cloud_init_content = ""
  requester_ip = module.requester_instance.requester_private_ips[0]
  compute_instance_count = var.compute_count
  compute_instance_type = var.compute_machine_type

}