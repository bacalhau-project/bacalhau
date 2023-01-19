provider "google" {
  project = var.gcp_project
  region  = var.region
  zone    = var.zone
}

terraform {
  backend "gcs" {
    # this bucket lives in the bacalhau-cicd google project
    # https://console.cloud.google.com/storage/browser/bacalhau-global-storage;tab=objects?project=bacalhau-cicd
    bucket = "bacalhau-global-storage"
    prefix = "terraform/dashboard/state"
  }
}

// A single Google Cloud Engine instance
resource "google_compute_instance" "dashboard_vm" {
  name         = "dashboard-vm-${terraform.workspace}-${count.index}"
  count        = 1
  machine_type = var.machine_type
  zone         = var.zone

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
      size  = var.boot_disk_size_gb
    }
  }

  metadata_startup_script = <<-EOF
#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

sudo mkdir -p /terraform_node
sudo tee /terraform_node/install-node.sh > /dev/null <<'EOI'
${file("${path.module}/remote_files/scripts/install-node.sh")}
EOI

sudo bash /terraform_node/install-node.sh 2>&1 | tee -a /tmp/bacalhau.log
EOF

  network_interface {
    network    = google_compute_network.dashboard_network[0].name
    subnetwork = ""
    access_config {
      nat_ip = google_compute_address.ipv4_address[count.index].address
    }
  }

  lifecycle {
    ignore_changes = [attached_disk]
  }
  #   service_account {
  #     scopes = ["cloud-platform"]
  #   }
  allow_stopping_for_update = true
}

resource "google_compute_address" "ipv4_address" {
  region = var.region
  name  = "bacalhau-dashboard-ipv4-address-${count.index}"
  count = 1
  lifecycle {
    prevent_destroy = true
  }
}

output "public_ip_address" {
  value = google_compute_instance.dashboard_vm.*.network_interface.0.access_config.0.nat_ip
}

resource "google_compute_disk" "dashboard_disk" {
  name     = "dashboard-disk-${terraform.workspace}-${count.index}"
  count    = 1
  type     = "pd-ssd"
  zone     = var.zone
  size     = var.volume_size_gb
  lifecycle {
    prevent_destroy = true
  }
}

resource "google_compute_disk_resource_policy_attachment" "attachment" {
  name  = google_compute_resource_policy.dashboard_disks_backups[count.index].name
  disk  = google_compute_disk.dashboard_disk[count.index].name
  zone  = var.zone
  count = 1
}

resource "google_compute_resource_policy" "dashboard_disks_backups" {
  name   = "dashboard-disk-backups-${terraform.workspace}-${count.index}"
  region = var.region
  count  = 1
  snapshot_schedule_policy {
    schedule {
      daily_schedule {
        days_in_cycle = 1
        start_time    = "23:00"
      }
    }
    retention_policy {
      max_retention_days    = 30
      on_source_disk_delete = "KEEP_AUTO_SNAPSHOTS"
    }
    snapshot_properties {
      labels = {
        dashboard_backup = "true"
      }
      # this only works with Windows and looks like it's non-negotiable with gcp
      guest_flush = false
    }
  }
}

resource "google_compute_attached_disk" "default" {
  disk     = google_compute_disk.dashboard_disk[count.index].self_link
  instance = google_compute_instance.dashboard_vm[count.index].self_link
  count    = 1
  zone     = var.zone
}

resource "google_compute_firewall" "dashboard_firewall" {
  name    = "dashboard-ingress-firewall-${terraform.workspace}"
  network = google_compute_network.dashboard_network[0].name

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports = [
      "80",
      "443"
    ]
  }

  source_ranges = var.ingress_cidrs
}

resource "google_compute_firewall" "dashboard_ssh_firewall" {
  name    = "dashboard-ssh-firewall-${terraform.workspace}"
  network = google_compute_network.dashboard_network[0].name

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    // Port 22   - Provides ssh access to the bacalhau server, for debugging 
    ports = ["22"]
  }

  source_ranges = var.ssh_access_cidrs
}

resource "google_compute_network" "dashboard_network" {
  name                    = "dashboard-network-${terraform.workspace}"
  auto_create_subnetworks = true
  count                   = 1
}
