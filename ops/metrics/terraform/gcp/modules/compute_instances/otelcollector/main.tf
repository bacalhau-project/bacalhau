resource "google_compute_instance" "otel_collector" {
  name         = "bacalhau-otel-collector"
  machine_type = var.otel_collector_instance_type
  zone         = var.zone

  metadata = {
    startup-script = local.otel_start_script
    user-data = data.cloudinit_config.otel_collector_cloud_init.rendered
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
      // TODO here is where we may wish to assign a static IP so this instance can be fronted with DNS
      // Ephemeral public IP will be assigned
    }
  }
}

locals {
  //
  // templating the bacalhau start script
  //
  otel_start_script = templatefile("${path.module}/../../../../instance_files/start.sh", {
    // Add more arguments as needed
  })

  //
  // templating otel config file
  //
  otel_config_content = templatefile("${path.module}/../../../../instance_files/otel-collector-config.yaml", {
    grafana_prometheus_username = var.grafana_prometheus_username
    grafana_prometheus_password = var.grafana_prometheus_password
  })

  //
  // templating otel service file
  //
  otel_service_content = templatefile("${path.module}/../../../../instance_files/otel.service", {
    // add more arguments as needed
  })

}

data "cloudinit_config" "otel_collector_cloud_init" {
  gzip        = false
  base64_encode = false

  // provide parameters to cloud-init like files and arguments to scripts in the above part.
  part {
    filename     = "cloud-config.yaml"
    content_type = "text/cloud-config"

    content = templatefile("${path.module}/../../../../cloud-init/cloud-init.yml", {
      otel_config_file      : base64encode(local.otel_config_content)
      otel_service_file     : base64encode(local.otel_service_content),
    })
  }
}
