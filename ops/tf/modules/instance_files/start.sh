#!/bin/bash

set -x

NODE_TYPE="${node_type}"
BACALHAU_VERSION_CMD="${bacalhau_version_cmd}"

# mount or format repo disk
function setup-bacalhau-repo-disk() {
    # Check if /data is already mounted
    if ! mountpoint -q /data; then
        # Check if disk has a filesystem
        if sudo blkid /dev/disk/by-id/google-bacalhau-repo; then
            echo "Repo disk already formatted, mounting..."
        else
            echo "Formatting repo disk..."
            sudo mkfs.ext4 /dev/disk/by-id/google-bacalhau-repo
        fi
        echo "Mounting repo disk..."
        sudo mkdir -p /data
        sudo mount /dev/disk/by-id/google-bacalhau-repo /data
        echo "/dev/disk/by-id/google-bacalhau-repo /data ext4 defaults,nofail 0 2" | sudo tee -a /etc/fstab
    fi
}

function setup-bacalhau-local-disk() {
    # Check if /local_data is already mounted
    if ! mountpoint -q /local_data; then
        # Check if disk has a filesystem
        if sudo blkid /dev/disk/by-id/google-bacalhau-local; then
            echo "Local data disk already formatted, mounting..."
        else
            echo "Formatting local data disk..."
            sudo mkfs.ext4 /dev/disk/by-id/google-bacalhau-local
        fi
        echo "Mounting local data disk..."
        sudo mkdir -p /local_data
        sudo mount /dev/disk/by-id/google-bacalhau-local /local_data
        echo "/dev/disk/by-id/google-bacalhau-local /local_data ext4 defaults,nofail 0 2" | sudo tee -a /etc/fstab
    fi

}

function setup-bacalhau-config() {
  echo "Moving bacalhau config to repo..."
  sudo mv /etc/config.yaml /data/config.yaml
}

function install-otel-collector() {
    wget "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v0.92.0/otelcol-contrib_0.92.0_linux_386.tar.gz"
    tar xvf "otelcol-contrib_0.92.0_linux_386.tar.gz"
    sudo mv otelcol-contrib /usr/local/bin/otelcol
}

function install-bacalhau() {
    echo "Installing bacalhau"
    export HOME=/root
    export GOCACHE="$HOME/.cache/go-build"
    export GOPATH="/root/go"
    bash /etc/install-bacalhau.sh $BACALHAU_VERSION_CMD
}

# reload service files and enable services
function setup-services() {
  echo "Loading systemctl services..."
  sudo systemctl daemon-reload
  echo "Enabling systemctl services..."
  sudo systemctl enable docker
  sudo systemctl enable otel.service
  sudo systemctl enable bacalhau.service
}

# start services
function start-services() {
  echo "Starting systemctl services..."
  sudo systemctl restart docker
  sudo systemctl restart otel.service
  sudo systemctl restart bacalhau.service
}

# setup and start everything
function start() {
  echo "Starting..."
  setup-bacalhau-repo-disk

  if [ "$NODE_TYPE" == "compute" ]; then
    setup-bacalhau-local-disk
  fi

  if [ "$BACALHAU_VERSION_CMD" != "" ]; then
    install-bacalhau
  fi

  # TODO move this into the VMI, maybe?
  install-otel-collector
  setup-bacalhau-config
  setup-services
  start-services
}

start &> /var/log/startup-script.log
