provider "google" {
  project = var.gcp_project
  region  = var.region
  zone    = var.zone
}

// A single Google Cloud Engine instance
resource "google_compute_instance" "bacalhau_vm" {
  name         = "bacalhau-vm-${count.index}"
  count        = var.instance_count
  machine_type = var.machine_type
  zone         = var.zone

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
    }
  }

  metadata_startup_script = <<-EOF
#!/bin/bash -xe

sudo apt-get update && sudo apt-get install -y docker.io

# TODO: move this into two systemd units!

wget https://github.com/filecoin-project/bacalhau/releases/download/v0.1.7/bacalhau_v0.1.7_amd64.tar.gz
tar xfv bacalhau_v0.1.7_amd64.tar.gz
sudo mv ./bacalhau /usr/local/bin/bacalhau

wget https://dist.ipfs.io/go-ipfs/v0.12.2/go-ipfs_v0.12.2_linux-amd64.tar.gz
tar -xvzf go-ipfs_v0.12.2_linux-amd64.tar.gz
cd go-ipfs
sudo bash install.sh
ipfs --version

# wait for /dev/sdb to exist
while [ ! -e /dev/sdb ]; do
  sleep 1
  echo "waiting for /dev/sdb to exist"
done

# mount /dev/sdb at /data
sudo mkdir -p /data
sudo mount /dev/sdb /data || (sudo mkfs -t ext4 /dev/sdb && sudo mount /dev/sdb /data)

sudo mkdir -p /data/ipfs
export IPFS_PATH=/data/ipfs

ipfs init

(ipfs daemon \
    2>&1 >> /tmp/ipfs.log) &

export LOG_LEVEL=debug
export BACALHAU_PATH=/data

(while true; do bacalhau serve --ipfs-connect /ip4/127.0.0.1/tcp/5001 --port 1235 2>&1 || true; sleep 1; done \
        >> /tmp/bacalhau.log) &

EOF
  network_interface {
    network = google_compute_network.bacalhau_network.name

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
  name         = "bacalhau-ipv4-address-${count.index}"
  count        = var.instance_count
}

resource "google_compute_disk" "bacalhau_disk" {
  name     = "bacalhau-disk-${count.index}"
  count    = var.instance_count
  type     = "pd-ssd"
  zone     = var.zone
  size     = var.volume_size_gb
  snapshot = var.restore_from_backup

  lifecycle {
    prevent_destroy = true
  }

}

resource "google_compute_disk_resource_policy_attachment" "attachment" {
  name = google_compute_resource_policy.bacalhau_disk_backups.name
  disk = google_compute_disk.bacalhau_disk[count.index].name
  zone = var.zone
  count = var.instance_count
}

resource "google_compute_resource_policy" "bacalhau_disk_backups" {
  name   = "bacalhau-disk-backups"
  region = var.region
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
        bacalhau_backup = "true"
      }
      guest_flush = true
    }
  }
}

resource "google_compute_attached_disk" "default" {
  disk     = google_compute_disk.bacalhau_disk[count.index].self_link
  instance = google_compute_instance.bacalhau_vm[count.index].self_link
  count    = var.instance_count
}

resource "google_compute_firewall" "bacalhau_firewall" {
  name    = "bacalhau-ingress-firewall"
  network = google_compute_network.bacalhau_network.name

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports = [
        "4001", // ipfs swarm
        "5001", // ipfs API
        "1234", // bacalhau API
        "1235", // bacalhau swarm
    ]
  }

  source_ranges = var.ingress_cidrs
}

resource "google_compute_firewall" "bacalhau_ssh_firewall" {
  name    = "bacalhau-ssh-firewall"
  network = google_compute_network.bacalhau_network.name

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

resource "google_compute_network" "bacalhau_network" {
  name = "bacalhau-network"
}
