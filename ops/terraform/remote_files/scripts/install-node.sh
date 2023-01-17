#!/bin/bash
# shellcheck disable=SC1091,SC2312
set -euo pipefail
IFS=$'\n\t'

source /terraform_node/variables

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
}

function install-gpu() {
  if [[ "${GPU_NODE}" = "true" ]]; then
    echo "Installing GPU drivers"
    distribution=$(. /etc/os-release;echo "${ID}${VERSION_ID}" | sed -e 's/\.//g') \
      && wget https://developer.download.nvidia.com/compute/cuda/repos/"${distribution}"/x86_64/cuda-keyring_1.0-1_all.deb \
      && sudo dpkg -i cuda-keyring_1.0-1_all.deb
    distribution=$(. /etc/os-release;echo "${ID}${VERSION_ID}") \
      && curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
      && curl -s -L https://nvidia.github.io/libnvidia-container/"${distribution}"/libnvidia-container.list | \
            sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
            sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list

    sudo apt-get update && sudo apt-get install -y \
      linux-headers-"$(uname -r)" \
      cuda-drivers \
      nvidia-docker2
    sudo systemctl restart docker
    nvidia-smi # No idea why we have to run this once, but we do. Only then does nvidia-container-cli work.
  else
    echo "Not installing GPU drivers because GPU_NODE=${GPU_NODE}"
  fi
}

# Lay down a very basic web server to report when the node is healthy
function install-healthcheck() {
  sudo apt-get -y install --no-install-recommends wget gnupg ca-certificates
  wget -O - https://openresty.org/package/pubkey.gpg | sudo apt-key add -
  echo "deb http://openresty.org/package/ubuntu $(lsb_release -sc) main" \
      | sudo tee /etc/apt/sources.list.d/openresty.list
  sudo apt-get update -y
  sudo apt-get -y install --no-install-recommends openresty
  sudo cp /terraform_node/nginx.conf /usr/local/openresty/nginx/conf/nginx.conf
}

function install-ipfs() {
  wget "https://dist.ipfs.tech/go-ipfs/${IPFS_VERSION}/go-ipfs_${IPFS_VERSION}_linux-amd64.tar.gz"
  tar -xvzf "go-ipfs_${IPFS_VERSION}_linux-amd64.tar.gz"
  # TODO should reset PWD to home dir after each function call
  cd go-ipfs
  sudo bash install.sh
  ipfs --version
}

function install-bacalhau() {
  wget "https://github.com/filecoin-project/bacalhau/releases/download/${BACALHAU_VERSION}/bacalhau_${BACALHAU_VERSION}_linux_amd64.tar.gz"
  tar xfv "bacalhau_${BACALHAU_VERSION}_linux_amd64.tar.gz"
  sudo mv ./bacalhau /usr/local/bin/bacalhau
}

function install-prometheus() {
  if [[ -z "${PROMETHEUS_VERSION}" ]] || [[ -z "${GRAFANA_CLOUD_API_ENDPOINT}" ]] || [[ -z "${GRAFANA_CLOUD_API_USER}" ]] || [[ -z "${GRAFANA_CLOUD_API_KEY}" ]]; then
    echo 'PROMETHEUS_VERSION or any of the GRAFANA_CLOUD_API_* env variables are undefined. Skipping Prometheus installation.'
  else
    sudo apt -y update
    sudo groupadd --system prometheus
    sudo useradd -s /sbin/nologin --system -g prometheus prometheus
    sudo mkdir -p /etc/prometheus
    sudo mkdir -p /var/lib/prometheus
    wget "https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/prometheus-${PROMETHEUS_VERSION}.linux-amd64.tar.gz"
    tar xvf "prometheus-${PROMETHEUS_VERSION}.linux-amd64.tar.gz"
    # TODO should reset PWD to home dir after each function call
    cd "prometheus-${PROMETHEUS_VERSION}.linux-amd64"
    sudo mv prometheus promtool /usr/local/bin/
    sudo mv consoles/ console_libraries/ /etc/prometheus/
    # config file
    HOSTNAME=$(hostname)
    sudo tee /terraform_node/prometheus.yml > /dev/null <<EOF
        global:
          scrape_interval: 240s
          evaluation_interval: 60s
          external_labels:
            origin_prometheus: ${HOSTNAME}

        scrape_configs:
          - job_name: 'opentelemetry'
            static_configs:
              - targets: ['localhost:2112']

        remote_write:
        - url: ${GRAFANA_CLOUD_API_ENDPOINT}
          basic_auth:
            username: ${GRAFANA_CLOUD_API_USER}
            password: ${GRAFANA_CLOUD_API_KEY}
EOF
    sudo cp /terraform_node/prometheus.yml /etc/prometheus/prometheus.yml
    sudo chown -R prometheus:prometheus /var/lib/prometheus/
  fi
}

function install-promtail() {
  if [[ -z "${LOKI_VERSION}" ]] || [[ -z "${GRAFANA_CLOUD_API_KEY}" ]] || [[ -z "${GRAFANA_CLOUD_LOKI_USER}" ]] || [[ -z "${GRAFANA_CLOUD_LOKI_ENDPOINT}" ]]; then
    echo 'Any of LOKI_VERSION, GRAFANA_CLOUD_API_KEY, GRAFANA_CLOUD_LOKI_USER, GRAFANA_CLOUD_LOKI_ENDPOINT env variables is undefined. Skipping Promtail/Loki installation.'
  else
    cd ~
    curl -O -L "https://github.com/grafana/loki/releases/download/v${LOKI_VERSION}/promtail-linux-amd64.zip"
    tar xvf promtail-linux-amd64.zip
    sudo chmod a+x "promtail-linux-amd64"
    sudo mv promtail-linux-amd64 /usr/local/bin/
    
    # config file
    HOSTNAME=$(hostname)
    sudo tee /terraform_node/promtail.yml > /dev/null <<EOF
server:
  http_listen_port: 0
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: https://${GRAFANA_CLOUD_LOKI_USER}:${SECRETS_GRAFANA_CLOUD_API_KEY}@${GRAFANA_CLOUD_LOKI_ENDPOINT}/loki/api/v1/push

scrape_configs:
  - job_name: journal
    journal:
      max_age: 12h
      labels:
        job: systemd-journal
        host: ${HOSTNAME}
        label_project: bacalhau
    relabel_configs:
      - action: keep
        source_labels: [__journal__systemd_unit]
        regex: '^bacalhau-daemon\.service$'
      - source_labels: ['__journal__systemd_unit']
        target_label: 'systemd_unit'
EOF
    sudo mkdir -p /etc/promtail
    sudo cp /terraform_node/promtail.yml /etc/promtail/config.yml
  fi
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

# make sure that "ipfs init" has been run
function init-ipfs() {
  sudo mkdir -p /data/ipfs
  export IPFS_PATH=/data/ipfs

  if [[ ! -e /data/ipfs/version ]]; then
    ipfs init
  fi
}

# install any secrets provided as terraform vars
function install-secrets() {
  # set defaults
  export HONEYCOMB_KEY=""
  export HONEYCOMB_DATASET="bacalhau_production"
  export GRAFANA_CLOUD_API_KEY=""
  export ESTUARY_API_KEY=""
  if [[ -e /data/secrets.sh ]]; then
    source /data/secrets.sh
  fi

  # load new values if they were provided
  if [[ -n "${SECRETS_HONEYCOMB_KEY}" ]]; then
    export HONEYCOMB_KEY="${SECRETS_HONEYCOMB_KEY}"
  fi
  if [[ -n "${SECRETS_GRAFANA_CLOUD_API_KEY}" ]]; then
    export GRAFANA_CLOUD_API_KEY="${SECRETS_GRAFANA_CLOUD_API_KEY}"
  fi
  if [[ -n "${SECRETS_ESTUARY_API_KEY}" ]]; then
    export ESTUARY_API_KEY="${SECRETS_ESTUARY_API_KEY}"
  fi

  # write the secrets to persistent disk
  sudo tee /data/secrets.sh > /dev/null <<EOG
export HONEYCOMB_KEY="${HONEYCOMB_KEY}"
export HONEYCOMB_DATASET="${HONEYCOMB_DATASET}"
export GRAFANA_CLOUD_API_KEY="${GRAFANA_CLOUD_API_KEY}"
export ESTUARY_API_KEY="${ESTUARY_API_KEY}"
EOG

  # clean up variables file from any secret
  sed -e '/^export SECRETS_/d' /terraform_node/variables | sudo tee /terraform_node/variables > /dev/null
}

# if we are node zero, are in unsafe mode and don't have a private key
# then let's copy the unsafe private key so we have a deterministic id
# that other nodes will connect to
function init-bacalhau() {
  export BACALHAU_NODE_PRIVATE_KEY_PATH="/data/.bacalhau/private_key.${BACALHAU_PORT}"
  sudo mkdir -p /data/.bacalhau
  if [[ "${TERRAFORM_NODE_INDEX}" == "0" ]] && [[ -n "${BACALHAU_UNSAFE_CLUSTER}" ]] && [[ ! -f "${BACALHAU_NODE_PRIVATE_KEY_PATH}" ]]; then
    echo "WE ARE NOW INSTALLING THE UNSAFE KEY YO"
    sudo cp /terraform_node/bacalhau-unsafe-private-key "${BACALHAU_NODE_PRIVATE_KEY_PATH}"
    sudo chmod 0600 "${BACALHAU_NODE_PRIVATE_KEY_PATH}"
  fi
}

function start-services() {
  sudo systemctl daemon-reload
  sudo systemctl enable ipfs-daemon
  sudo systemctl enable bacalhau-daemon
  sudo systemctl enable prometheus-daemon
  sudo systemctl enable promtail
  sudo systemctl start ipfs-daemon
  sudo systemctl start bacalhau-daemon
  sudo systemctl start prometheus-daemon
  sudo systemctl start promtail
  sudo service openresty reload
}

function install() {
  install-docker
  install-gpu
  install-healthcheck
  install-ipfs
  install-bacalhau
  mount-disk
  init-ipfs
  init-bacalhau
  install-secrets
  install-prometheus
  install-promtail
  start-services
}

install
