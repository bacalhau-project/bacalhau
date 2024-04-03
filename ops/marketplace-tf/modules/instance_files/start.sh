#!/bin/bash

set -x

BACALHAU_VERSION_CMD="${bacalhau_version_cmd}"

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

  if [ "$BACALHAU_VERSION_CMD" != "" ]; then
    install-bacalhau
  fi

  # TODO move this into the VMI, maybe?
  install-otel-collector
  setup-services
  start-services
}

start &> /var/log/startup-script.log
