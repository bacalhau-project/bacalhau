provider "google" {
  project = var.gcp_project
  region  = var.region
  zone    = var.zone
}

terraform {
  backend "gcs" {
    # this bucket lives in the bacalhau-cicd google projecty
    # https://console.cloud.google.com/storage/browser/bacalhau-global-storage;tab=objects?project=bacalhau-cicd
    bucket = "bacalhau-global-storage"
    prefix = "terraform/state"
  }
}

// A single Google Cloud Engine instance
resource "google_compute_instance" "bacalhau_vm" {
  name         = "bacalhau-vm-${terraform.workspace}-${count.index}"
  count        = var.instance_count
  machine_type = var.machine_type
  zone         = var.zone

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
      size = var.boot_disk_size_gb
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

sudo mkdir -p /var/www/health_checker

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

sudo tee /etc/systemd/system/ipfs-daemon.service > /dev/null <<'EOI'
${file("${path.module}/configs/ipfs-daemon.service")}
EOI

sudo tee /etc/systemd/system/bacalhau-daemon.service > /dev/null <<'EOI'
${file("${path.module}/configs/bacalhau-daemon.service")}
EOI

# the path to the private key on the node
export BACALHAU_NODE_PRIVATE_KEY_PATH="/data/.bacalhau/private_key.${var.bacalhau_port}"

# should we write the unsafe private key to node0 if we are in auto connect mode
export BACALHAU_SHOULD_WRITE_NODE0_UNSAFE_KEY="${var.bacalhau_unsafe_cluster && count.index == 0 ? "yes" : ""}"
sudo mkdir -p /data/.bacalhau

if [ -n "$BACALHAU_SHOULD_WRITE_NODE0_UNSAFE_KEY" ]; then
  # only write the unsafe key if there is not already one
  if [ ! -f "$BACALHAU_NODE_PRIVATE_KEY_PATH" ]; then
    sudo tee "$BACALHAU_NODE_PRIVATE_KEY_PATH" > /dev/null <<'EOI'
${file("${path.module}/configs/unsafe-private-key")}
EOI
    sudo chmod 0600 "$BACALHAU_NODE_PRIVATE_KEY_PATH"
  fi
fi

# we need this as a script so we can write some terraform variables
# into the startup script that is then called by systemd
sudo tee /start-bacalhau.sh > /dev/null <<'EOI'
#!/bin/bash
set -euo pipefail
IFS=$'\n\t'
# the ip of node0
export BACALHAU_NODE0_IP="${google_compute_address.ipv4_address[0].address}"
# calculate what the unsafe id would be - either the fixed unsafe id or empty string
# this just avoids nested terraform ternary expressions
export BACALHAU_NODE0_UNSAFE_ID="${var.bacalhau_unsafe_cluster ? "QmUqesBmpC7pSzqH86ZmZghtWkLwL6RRop3M1SrNbQN5QD" : ""}"
export BACALHAU_NODE0_ID="${var.bacalhau_connect_node0 != "" ? var.bacalhau_connect_node0 : "$BACALHAU_NODE0_UNSAFE_ID"}"
# the fully exploded multiaddress for node0
export BACALHAU_NODE0_MULTIADDRESS="/ip4/$BACALHAU_NODE0_IP/tcp/${var.bacalhau_port}/p2p/$BACALHAU_NODE0_ID"
# work out if we actually want to connect to that multiaddress
# if we are > node0 and have either an explicit node0 id or are in unsafe mode - then we do want to connect
export BACALHAU_CONNECT_PEER="${count.index > 0 && (var.bacalhau_connect_node0 || var.bacalhau_short_lived_cluster) ? "$BACALHAU_NODE0_MULTIADDRESS" : "none"}"
bacalhau serve \
  --job-selection-data-locality anywhere \
  --ipfs-connect /ip4/127.0.0.1/tcp/5001 \
  --port ${var.bacalhau_port} \
  --peer $BACALHAU_CONNECT_PEER
EOI

# activate the systemd units now
sudo systemctl daemon-reload
sudo systemctl enable ipfs-daemon.service
sudo systemctl enable bacalhau-daemon.service
sudo systemctl start ipfs-daemon
sudo systemctl start bacalhau-daemon

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
    subnetwork = google_compute_subnetwork.bacalhau_subnetwork.name

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
  # keep the same ip addresses if we are production (because they are in DNS and the auto connect serve codebase)
  name  = terraform.workspace == "production" ? "bacalhau-ipv4-address-${count.index}" : "bacalhau-ipv4-address-${terraform.workspace}-${count.index}"
  count = var.instance_count
  lifecycle {
    prevent_destroy = true
  }
}

output "public_ip_address" {
  value = google_compute_instance.bacalhau_vm.*.network_interface.0.access_config.0.nat_ip
}

resource "google_compute_disk" "bacalhau_disk" {
  # keep the same disk names if we are production because the libp2p ids are in the auto connect serve codebase
  name     = terraform.workspace == "production" ? "bacalhau-disk-${count.index}" : "bacalhau-disk-${terraform.workspace}-${count.index}"
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
  name   = "bacalhau-disk-backups-${terraform.workspace}"
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
  name    = "bacalhau-ingress-firewall-${terraform.workspace}"
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
  name    = "bacalhau-ssh-firewall-${terraform.workspace}"
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

resource "google_compute_subnetwork" "bacalhau_subnetwork" {
  name          = "bacalhau-subnetwork-${terraform.workspace}"
  ip_cidr_range = "192.168.0.0/16"
  region        = var.region
  network       = google_compute_network.bacalhau_network.id
}

resource "google_compute_network" "bacalhau_network" {
  name = "bacalhau-network-${terraform.workspace}"
  auto_create_subnetworks = false
}
