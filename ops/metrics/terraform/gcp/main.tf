provider "google" {
  project = var.gcp_project_id
  region  = var.gcp_region
  zone = var.gcp_zone
}

module "gcp_network" {
  source = "./modules/network"
  region = var.gcp_region
  subnet_cidr = "10.0.0.0/16"
}

module "otel_collector_instance" {
  source = "./modules/compute_instances/otelcollector"

  cloud_init_content = ""

  zone                = var.gcp_zone
  network             = module.gcp_network.vpc_network_name
  subnetwork          = module.gcp_network.subnetwork_name

  boot_image                    = var.gcp_boot_image
  otel_collector_instance_type = var.otel_collector_machine_type

  grafana_prometheus_username = var.grafana_prometheus_username
  grafana_prometheus_password = var.grafana_prometheus_password
}