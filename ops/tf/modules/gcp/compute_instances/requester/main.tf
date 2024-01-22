// define requester instance
resource "google_compute_instance" "requester" {
  name         = "bacalhau-requester"
  machine_type = var.requester_instance_type
  zone         = var.zone

  metadata = {
    startup-script = local.bacalhau_start_script
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

// define disk to contain the bacalhau repo for instance
resource "google_compute_disk" "bacalhau_repo_disks" {
  name  = "bacalhau-repo-disk-requester"
  type  = "pd-standard"
  zone  = var.zone
  size  = var.bacalhau_repo_disk_size
}

// attach the disk to instance
resource "google_compute_attached_disk" "attach_bacalhau_repo_disks" {
  disk     = google_compute_disk.bacalhau_repo_disks.self_link
  instance = google_compute_instance.requester.self_link
  device_name = "bacalhau-repo"
}

locals {
  //
  // templating the bacalhau service file
  //

  // service env vars
  bacalhau_env_vars = {
    LOG_LEVEL                 = "debug"
    BACALHAU_NODE_LOGGINGMODE = "default"
    BACALHAU_DIR              = "/data"
    BACALHAU_ENVIRONMENT      = "local"
    AWS_ACCESS_KEY_ID         = var.aws_access_key_id
    AWS_SECRET_ACCESS_KEY     = var.aws_secret_access_key
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
  requester_config_content = templatefile("${path.module}/../../../instance_files/requester_config.yaml", {
    # add variables you'd like to inject into the config
    bacalhau_accept_networked_jobs = var.bacalhau_accept_networked_jobs
  })

  //
  // templating the bacalhau start script
  //
  bacalhau_start_script = templatefile("${path.module}/../../../instance_files/start.sh", {
    node_type = "requester"
    // Add more arguments as needed
  })
}


data "cloudinit_config" "requester_cloud_init" {
  gzip        = false
  base64_encode = false

  // provide parameters to cloud-init like files and arguments to scripts in the above part.
  part {
    filename     = "cloud-config.yaml"
    content_type = "text/cloud-config"

    content = templatefile("${path.module}/../../../cloud-init/cloud-init.yml", {
      bacalhau_config_file  : base64encode(local.requester_config_content),
      bacalhau_service_file : base64encode(local.bacalhau_service_content)
    })
  }
}