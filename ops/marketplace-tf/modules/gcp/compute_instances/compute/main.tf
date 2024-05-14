// define compute instance(s)
resource "google_compute_instance" "compute" {
  count = var.compute_instance_count
  name         = "bacalhau-compute-${count.index + 1}"
  machine_type = var.compute_instance_type
  zone         = var.gcp_config.zone

  metadata = {
    user-data = data.cloudinit_config.compute_cloud_init.rendered
  }

  metadata_startup_script = local.bacalhau_start_script

  boot_disk {
    initialize_params {
      image = var.gcp_config.boot_image
      size = var.disk_config.boot_size
    }
  }

  lifecycle {
    ignore_changes = [attached_disk]
  }
  allow_stopping_for_update = true

  network_interface {
    network = var.gcp_config.network
    subnetwork = var.gcp_config.subnetwork
    access_config {
      // Ephemeral public IP will be assigned
    }
  }
}

// define disk(s) to contain the bacalhau repo for instance(s)
resource "google_compute_disk" "bacalhau_repo_disks" {
  count = var.compute_instance_count
  name  = "bacalhau-repo-disk-compute-${count.index + 1}"
  type  = "pd-standard"
  zone  = var.gcp_config.zone
  size  = var.disk_config.repo_size
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
  zone  = var.gcp_config.zone
  size  = var.disk_config.local_size
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
  // templating the bacalhau install script file

  // service env vars
  bacalhau_env_vars = {
    LOG_LEVEL                   = "debug"
    BACALHAU_NODE_LOGGINGMODE   = "default"
    BACALHAU_DIR                = "/bacalhau_repo"
    BACALHAU_ENVIRONMENT        = "local"
    // TODO make this a variable
    OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4318"
    AWS_ACCESS_KEY_ID           = var.aws_credentials.access_key_id
    AWS_SECRET_ACCESS_KEY       = var.aws_credentials.secret_access_key
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
    compute_api_token = var.token_config.compute_api_token
  })

  //
  // templating the bacalhau start script
  //

  // inject custom bacalhau install based on variables.
  // I am sorry reader, terraform requires this be one line
  bacalhau_install_cmd_content = var.build_config.install_version  != "" ? "release ${var.build_config.install_version}" : var.build_config.install_branch  != "" ? "branch ${var.build_config.install_branch}" : var.build_config.install_commit  != "" ?"commit ${var.build_config.install_commit}" : ""
  bacalhau_start_script = templatefile("${path.module}/../../../instance_files/start.sh", {
    bacalhau_version_cmd = local.bacalhau_install_cmd_content
    // Add more arguments as needed
  })

  bacalhau_install_script_content = file("${path.module}/../../../instance_files/install-bacalhau.sh")

  //
  // templating otel config file
  //
  otel_config_content = templatefile("${path.module}/../../../instance_files/otel-collector.yaml", {
    bacalhau_otel_collector_endpoint = var.bacalhau_otel_collector_endpoint
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

    content = templatefile("${path.module}/../../../cloud-init/compute-cloud-init.yml", {
      bacalhau_install_script_file: base64encode(local.bacalhau_install_script_content)
      bacalhau_config_file        : base64encode(local.compute_config_content)
      bacalhau_service_file       : base64encode(local.bacalhau_service_content)
      otel_config_file            : base64encode(local.otel_config_content)
      otel_service_file           : base64encode(local.otel_service_content)
      requester_ip                : var.requester_ip
      tls_cert_file               : base64encode(var.bacalhau_tls_crt)
    })
  }
}