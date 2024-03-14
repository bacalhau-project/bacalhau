#!/bin/bash

function install-otel-collector() {
    wget "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v0.92.0/otelcol-contrib_0.92.0_linux_386.tar.gz"
    tar xvf "otelcol-contrib_0.92.0_linux_386.tar.gz"
    sudo mv otelcol-contrib /usr/local/bin/otelcol
}

# reload service files and enable services
function setup-services() {
  echo "Loading systemctl services..."
  sudo systemctl daemon-reload
  echo "Enabling systemctl services..."
  sudo systemctl enable otel.service
}

# start services
function start-services() {
  echo "Starting systemctl services..."
  sudo systemctl restart otel.service
}

# setup and start everything
function start() {
  echo "Starting..."
  # TODO move this into the VMI, maybe?
  install-otel-collector
  setup-services
  start-services
}

start &> /var/log/startup-script.log
