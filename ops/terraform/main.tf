provider "google" {
  project = var.gcp_project
  region  = var.region
  zone    = var.zone
}

terraform {
  backend "gcs" {
    bucket = "bacalhau-global-storage"
    prefix = "terraform/state"
  }
}

// A single Google Cloud Engine instance
resource "google_compute_instance" "bacalhau_vm" {
  name         = "bacalhau-vm-${var.rollout_phase}-${count.index}"
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

sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update -y
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
# TODO: move this into two systemd units!

sudo mkdir -p /var/www/health_checker

# Lay down a very basic web server to report when the node is healthy
sudo apt-get -y install --no-install-recommends wget gnupg ca-certificates
wget -O - https://openresty.org/package/pubkey.gpg | sudo apt-key add -
echo "deb http://openresty.org/package/ubuntu $(lsb_release -sc) main" \
    | sudo tee /etc/apt/sources.list.d/openresty.list
sudo apt-get update -y
sudo apt-get -y install --no-install-recommends openresty

sudo tee /usr/local/openresty/nginx/conf/nginx.conf > /dev/null <<'EOI'
${file("${path.module}/configs/nginx.conf")}
EOI

sudo tee /var/www/health_checker/livez.sh > /dev/null <<'EOI'
${file("${path.module}/scripts/livez.sh")}
EOI

sudo tee /var/www/health_checker/healthz.sh > /dev/null <<'EOI'
${file("${path.module}/scripts/healthz.sh")}
EOI

sudo chmod u+x /var/www/health_checker/*.sh

wget https://github.com/filecoin-project/bacalhau/releases/download/${var.bacalhau_version}/bacalhau_${var.bacalhau_version}_linux_amd64.tar.gz
tar xfv bacalhau_${var.bacalhau_version}_linux_amd64.tar.gz
sudo mv ./bacalhau /usr/local/bin/bacalhau

wget https://dist.ipfs.io/go-ipfs/${var.ipfs_version}/go-ipfs_${var.ipfs_version}_linux-amd64.tar.gz
tar -xvzf go-ipfs_${var.ipfs_version}_linux-amd64.tar.gz
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

if [ ! -e /data/ipfs/version ]; then
  ipfs init
fi

(ipfs daemon \
    2>&1 >> /tmp/ipfs.log) &

export LOG_LEVEL=debug
export BACALHAU_PATH=/data

(while true; do bacalhau serve --job-selection-data-locality anywhere --peer ${count.index == 0 ? "none" : "/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL"} --ipfs-connect /ip4/127.0.0.1/tcp/5001 --port 1235 2>&1 || true; sleep 1; done \
        >> /tmp/bacalhau.log) &

sudo service openresty reload
sudo tee /var/www/health_checker/network_name.txt > /dev/null <<EOI
${google_compute_network.bacalhau_network.name}
EOI

sudo tee /var/www/health_checker/address.txt > /dev/null <<EOI
${google_compute_address.ipv4_address[count.index].address}
EOI

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
  name  = "bacalhau-ipv4-address-${var.rollout_phase}-${count.index}"
  count = var.instance_count
}

output "public_ip_address" {
  value = google_compute_instance.bacalhau_vm.*.network_interface.0.access_config.0.nat_ip
}

resource "google_compute_disk" "bacalhau_disk" {
  name     = "bacalhau-disk-${var.rollout_phase}-${count.index}"
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
  name  = google_compute_resource_policy.bacalhau_disk_backups.name
  disk  = google_compute_disk.bacalhau_disk[count.index].name
  zone  = var.zone
  count = var.instance_count
}

resource "google_compute_resource_policy" "bacalhau_disk_backups" {
  name   = "bacalhau-disk-backups-${var.rollout_phase}"
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
  name    = "bacalhau-ingress-firewall-${var.rollout_phase}"
  network = google_compute_network.bacalhau_network.name

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports = [
      "4001",  // ipfs swarm
      "5001",  // ipfs API
      "1234",  // bacalhau API
      "1235",  // bacalhau swarm
      "44443", // nginx is healthy - for running health check scripts
      "44444", // nginx node health check scripts
    ]
  }

  source_ranges = var.ingress_cidrs
}

resource "google_compute_firewall" "bacalhau_ssh_firewall" {
  name    = "bacalhau-ssh-firewall-${var.rollout_phase}"
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
  name = "bacalhau-network-${var.rollout_phase}"
}
