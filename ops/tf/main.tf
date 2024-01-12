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

resource "google_compute_instance" "requester" {
  name         = "bacalhau-requester"
  machine_type = var.requester_machine_type
  zone         = var.gcp_zone

  metadata = {
    node_type = "requester"
  }

  boot_disk {
    initialize_params {
      image = "projects/forrest-dev-407420/global/images/bacalhau-ubuntu-2004-lts-test-19"
      size = 50
    }
  }

  network_interface {
    network = module.gcp_network.vpc_network_name
    subnetwork = module.gcp_network.subnetwork_name
    access_config {
      nat_ip = module.gcp_network.requester_ip
    }
  }

}

resource "google_compute_instance" "compute" {
  count = var.compute_count
  name         = "compute-instance"
  machine_type = var.compute_machine_type
  zone         = var.gcp_zone

  metadata = {
    node_type    = "compute"
    requester_ip = module.gcp_network.requester_ip
  }

  boot_disk {
    initialize_params {
      image = var.gcp_boot_image
    #   "projects/forrest-dev-407420/global/images/bacalhau-ubuntu-2004-lts-test-18"
      size = var.gcp_boot_disk_size
    }
  }

  network_interface {
    network = module.gcp_network.vpc_network_name
    subnetwork = module.gcp_network.subnetwork_name
    access_config {
      // Ephemeral public IP will be assigned
    }
  }

}
