provider "google" {
  project = var.gcp_project_id
  region  = var.gcp_region
}


module "requester_instance" {
  source = "./modules/gcp/compute_instances/requester"

  zone = var.gcp_zone
  cloud_init_content = ""
  requester_instance_type = var.requester_machine_type
}

module "compute_instance" {
  source = "./modules/gcp/compute_instances/compute"

  zone = var.gcp_zone
  cloud_init_content = ""
  requester_ip = module.requester_instance.requester_private_ips
  compute_instance_count = var.compute_count
  compute_instance_type = var.compute_machine_type

}