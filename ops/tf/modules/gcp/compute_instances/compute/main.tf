resource "google_compute_instance" "compute" {
  count = var.compute_instance_count
  name         = "bacalhau-compute-${count.index + 1}"
  machine_type = var.compute_instance_type
  zone         = var.zone

  metadata = {
    user-data = data.cloudinit_config.compute_cloud_init.rendered
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
      // Ephemeral public IP will be assigned
    }
  }
}

locals {
  compute_config_content = templatefile("${path.module}/../../../instance_files/compute_config.yaml", {
    requester_ip = var.requester_ip
  })
  bacalhau_service_content = templatefile("${path.module}/../../../instance_files/bacalhau.service", {
    args = "" # replace with your actual arguments
  })
}


data "cloudinit_config" "compute_cloud_init" {
  gzip        = false
  base64_encode = false

  part {
    filename     = "cloud-config.yaml"
    content_type = "text/cloud-config"

    content = templatefile("${path.module}/../../../cloud-init/cloud-init.yml", {
      bacalhau_config_file  : base64encode(local.compute_config_content),
      bacalhau_service_file : base64encode(local.bacalhau_service_content),
      requester_ip          : var.requester_ip,
    })
  }
}
