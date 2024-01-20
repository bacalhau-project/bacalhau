#!/bin/bash

# mount or format repo disk
function setup-bacalhau-disk() {
    # Check if /data is already mounted
    if ! mountpoint -q /data; then
        # Assuming /dev/sdb is the new disk
        # Check if /dev/sdb has a filesystem
        if sudo blkid /dev/sdb; then
            echo "Disk already formatted, mounting..."
        else
            echo "Formatting disk..."
            sudo mkfs.ext4 /dev/sdb
        fi
        echo "Mounting disk..."
        sudo mkdir -p /data
        sudo mount /dev/sdb /data
        echo "/dev/sdb /data ext4 defaults,nofail 0 2" | sudo tee -a /etc/fstab
    fi
}

function setup-bacalhau-config() {
  echo "Moving bacalhau config to repo..."
  sudo mv /etc/config.yaml /data/config.yaml
}

# reload service files and enable services
function setup-services() {
  echo "Loading systemctl services..."
  sudo systemctl daemon-reload
  echo "Enabling systemctl services..."
  sudo systemctl enable docker
  sudo systemctl enable bacalhau.service
}

# start services
function start-services() {
  echo "Starting systemctl services..."
  sudo systemctl restart docker
  sudo systemctl restart bacalhau.service
}

# setup and start everything
function start() {
  echo "Starting..."
  setup-bacalhau-disk
  setup-bacalhau-config
  setup-services
  start-services
}

start &> /var/log/startup-script.log
