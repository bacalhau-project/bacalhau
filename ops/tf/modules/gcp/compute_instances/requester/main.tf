resource "google_compute_instance" "requester" {
  name         = "bacalhau-requester"
  machine_type = var.requester_instance_type
  zone         = var.zone

  metadata_startup_script = templatefile("${path.module}/../../cloud-init/cloud-init.yml", {
    config_file = "${path.module}/../../instance_files/requester_config.yaml",
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
      // Ephemeral IP
    }
  }
}
