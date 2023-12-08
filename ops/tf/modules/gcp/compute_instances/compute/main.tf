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
    network = "default"
    access_config {
      // Ephemeral IP
    }
  }
}

data "cloudinit_config" "compute_cloud_init" {
  gzip        = false
  base64_encode = false

  part {
    filename     = "cloud-config.yaml"
    content_type = "text/cloud-config"

    content      = templatefile("${path.module}/../../../cloud-init/cloud-init.yml", {
      bacalhau_config_file    : filebase64("${path.module}/../../../instance_files/compute_config.yaml"),
      bacalhau_service_file   : filebase64("${path.module}/../../../instance_files/bacalhau.service"),
      requester_ip            : var.requester_ip,
    })
  }
}
