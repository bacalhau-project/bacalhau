// define requester instance
resource "google_compute_instance" "requester" {
  name         = "bacalhau-requester"
  machine_type = var.requester_instance_type
  zone         = var.gcp_config.zone

  metadata = {
    user-data = data.cloudinit_config.requester_cloud_init.rendered
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
      nat_ip = var.requester_static_ip // Static IP
    }
  }
}

// define disk to contain the bacalhau repo for instance
resource "google_compute_disk" "bacalhau_repo_disks" {
  name  = "bacalhau-repo-disk-requester"
  type  = "pd-standard"
  zone  = var.gcp_config.zone
  size  = var.disk_config.repo_size
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

  bacalhau_install_script_content = file("${path.module}/../../../instance_files/install-bacalhau.sh")

  //
  // templating the bacalhau config file
  //
  requester_config_content = templatefile("${path.module}/../../../instance_files/requester_config.yaml", {
    # add variables you'd like to inject into the config
    bacalhau_accept_networked_jobs = var.bacalhau_accept_networked_jobs
    compute_api_token = var.token_config.compute_api_token
    requester_ip = var.requester_static_ip
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

  //
  // templating otel config file
  //
  otel_config_content = templatefile("${path.module}/../../../instance_files/otel-collector.yaml", {
    bacalhau_otel_collector_endpoint = var.bacalhau_otel_collector_endpoint
  })

  //
  // templating otel service file
  //
  otel_service_content = templatefile("${path.module}/../../../instance_files/otel.service", {
    // add more arguments as needed
  })

  //
  // templating rego
  //

  // authn
  bacalhau_authn_policy_content = templatefile("${path.module}/../../../instance_files/authn_policy.rego", {
    bacalhau_secret_user_access_token = var.token_config.requester_api_token
  })
  // authz
  bacalhau_authz_policy_content = templatefile("${path.module}/../../../instance_files/authz_policy.rego", {
    // add more arguments as needed
  })

  tls_cert_content = var.tls_config.bacalhau_tls_crt
  tls_key_content = var.tls_config.bacalhau_tls_sk
}


data "cloudinit_config" "requester_cloud_init" {
  gzip        = false
  base64_encode = false

  // provide parameters to cloud-init like files and arguments to scripts in the above part.
  part {
    filename     = "cloud-config.yaml"
    content_type = "text/cloud-config"

    content = templatefile("${path.module}/../../../cloud-init/requester-cloud-init.yml", {
      bacalhau_install_script_file: base64encode(local.bacalhau_install_script_content)
      bacalhau_config_file        : base64encode(local.requester_config_content)
      bacalhau_service_file       : base64encode(local.bacalhau_service_content)
      bacalhau_authn_policy_file  : base64encode(local.bacalhau_authn_policy_content)
      bacalhau_authz_policy_file  : base64encode(local.bacalhau_authz_policy_content)
      otel_config_file            : base64encode(local.otel_config_content)
      otel_service_file           : base64encode(local.otel_service_content)
      tls_cert_file               : base64encode(local.tls_cert_content)
      tls_key_file                : base64encode(local.tls_key_content)

    })
  }
}