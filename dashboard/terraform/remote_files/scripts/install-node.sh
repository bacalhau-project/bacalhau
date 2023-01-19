#!/bin/bash
# shellcheck disable=SC1091,SC2312
set -euo pipefail
IFS=$'\n\t'

function install-docker() {
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
  sudo curl -L "https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
  sudo chmod +x /usr/local/bin/docker-compose
}

function mount-disk() { 
  # wait for /dev/sdb to exist
  while [[ ! -e /dev/sdb ]]; do
    sleep 1
    echo "waiting for /dev/sdb to exist"
  done
  # mount /dev/sdb at /data
  sudo mkdir -p /data
  sudo mount /dev/sdb /data || (sudo mkfs -t ext4 /dev/sdb && sudo mount /dev/sdb /data) 
}

function install() {
  install-docker
  mount-disk
}

install
