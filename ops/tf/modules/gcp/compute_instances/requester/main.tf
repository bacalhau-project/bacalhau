resource "google_compute_instance" "requester" {
  name         = "bacalhau-requester"
  machine_type = var.requester_instance_type
  zone         = var.zone

  metadata = {
    user-data = data.cloudinit_config.requester_cloud_init.rendered
  }
  boot_disk {
    initialize_params {
      image = var.boot_image
      size = var.boot_size
    }
  }

  network_interface {
    network = var.network
    subnetwork = var.subnetwork
    access_config {
      nat_ip = var.requester_static_ip // Static IP
    }
  }

}

locals {
  requester_config_content = templatefile("${path.module}/../../../instance_files/requester_config.yaml", {
    # add variables you'd like to inject into the config
  })
  bacalhau_service_content = templatefile("${path.module}/../../../instance_files/bacalhau.service", {
    args = "" # replace with your actual arguments
  })
}

data "cloudinit_config" "requester_cloud_init" {
  gzip        = false
  base64_encode = false

  part {
    filename     = "cloud-config.yaml"
    content_type = "text/cloud-config"

    content = templatefile("${path.module}/../../../cloud-init/cloud-init.yml", {
      bacalhau_config_file  : base64encode(local.requester_config_content),
      bacalhau_service_file : base64encode(local.bacalhau_service_content),
    })
  }
}
