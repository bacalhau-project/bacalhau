resource "google_compute_instance" "compute" {
  count = var.compute_instance_count
  name         = "bacalhau-compute-${count.index + 1}"
  machine_type = var.compute_instance_type
  zone         = var.zone

  metadata_startup_script = templatefile("${path.module}/../../cloud-init/cloud-init.yml", {
    config_file = "${path.module}/../../instance_files/compute_config.yaml",
    requester_ip = var.requester_ip
  })

  boot_disk {
    initialize_params {
      image = var.boot_image
      size = var.boot_size
    }
  }

  network_interface {
    network = "default"
    access_config {
      nat_ip = true
    }
  }
}
