// define compute instance(s)
resource "google_compute_instance" "compute" {
  count = var.compute_instance_count
  name         = "bacalhau-compute-${count.index + 1}"
  machine_type = var.compute_instance_type
  zone         = var.zone

  metadata = {
    startup-script = local.bacalhau_start_script
    user-data = data.cloudinit_config.compute_cloud_init.rendered
  }

  boot_disk {
    initialize_params {
      image = var.boot_image
      size = var.boot_size
    }
  }

  scheduling {
  on_host_maintenance = "TERMINATE"
  automatic_restart   = false
  }

  network_interface {
    network = var.network
    subnetwork = var.subnetwork
    access_config {
      // Ephemeral public IP will be assigned
    }
  }

  guest_accelerator {
  type  = var.accelerator
  count = var.accelerator_count
  }
}

// define disk(s) to contain the bacalhau repo for instance(s)
resource "google_compute_disk" "bacalhau_repo_disks" {
  count = var.compute_instance_count
  name  = "bacalhau-repo-disk-compute-${count.index + 1}"
  type  = "pd-standard"
  zone  = var.zone
  size  = var.bacalhau_repo_disk_size
}

// attach the disk(s) to instance(s)
resource "google_compute_attached_disk" "attach_bacalhau_repo_disks" {
  count = var.compute_instance_count
  disk     = google_compute_disk.bacalhau_repo_disks[count.index].self_link
  instance = google_compute_instance.compute[count.index].self_link
  device_name = "bacalhau-repo"
}

// define disk(s) to contain the bacalhau repo for instance(s)
resource "google_compute_disk" "bacalhau_local_disks" {
  count = var.compute_instance_count
  name  = "bacalhau-local-disk-compute-${count.index + 1}"
  type  = "pd-standard"
  zone  = var.zone
  size  = var.bacalhau_local_disk_size
}

// attach the disk(s) to instance(s)
resource "google_compute_attached_disk" "attach_bacalhau_local_disks" {
  count = var.compute_instance_count
  disk     = google_compute_disk.bacalhau_local_disks[count.index].self_link
  instance = google_compute_instance.compute[count.index].self_link
  device_name = "bacalhau-local"
}

locals {
  //
  // templating the bacalhau service file
  //

  // service env vars
  bacalhau_env_vars = {
    LOG_LEVEL                   = "debug"
    BACALHAU_NODE_LOGGINGMODE   = "default"
    BACALHAU_DIR                = "/data"
    BACALHAU_ENVIRONMENT        = "local"
    // TODO make this a variable
    OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4318"
    AWS_ACCESS_KEY_ID           = var.aws_access_key_id
    AWS_SECRET_ACCESS_KEY       = var.aws_secret_access_key
    # Add more variables here as needed
  }
  # Convert the map to the required string format for the systemd service file
  env_vars_string = join("\n", [for k, v in local.bacalhau_env_vars : "Environment=\"${k}=${v}\""])

  // service bacalhau arguments
  bacalhau_args = ""

  bacalhau_service_content = templatefile("${path.module}/../../../instance_files/bacalhau.service", {
    env_vars = local.env_vars_string
    args = local.bacalhau_args,
  })

  //
  // templating the bacalhau config file
  //
  compute_config_content = templatefile("${path.module}/../../../instance_files/compute_config.yaml", {
    requester_ip = var.requester_ip
    bacalhau_accept_networked_jobs = var.bacalhau_accept_networked_jobs
  })

  //
  // templating the bacalhau start script
  //
  bacalhau_start_script = templatefile("${path.module}/../../../instance_files/start.sh", {
    node_type = "compute"
    // Add more arguments as needed
  })

  //
  // templating otel config file
  //
  otel_config_content = templatefile("${path.module}/../../../instance_files/otel-collector.yaml", {
    // add more arguments as needed
  })

  //
  // templating otel service file
  //
  otel_service_content = templatefile("${path.module}/../../../instance_files/otel.service", {
    // add more arguments as needed
  })
}


data "cloudinit_config" "compute_cloud_init" {
  gzip        = false
  base64_encode = false

  // provide parameters to cloud-init like files and arguments to scripts in the above part.
  part {
    filename     = "cloud-config.yaml"
    content_type = "text/cloud-config"

    content = templatefile("${path.module}/../../../cloud-init/cloud-init.yml", {
      bacalhau_config_file  : base64encode(local.compute_config_content),
      bacalhau_service_file : base64encode(local.bacalhau_service_content),
      otel_config_file      : base64encode(local.otel_config_content)
      otel_service_file     : base64encode(local.otel_service_content),
      requester_ip          : var.requester_ip,
    })
  }
}