#!/bin/bash
# shellcheck disable=SC1091,SC2312
set -euo pipefail
IFS=$'\n\t'

source /terraform_node/variables

function install-go() {
  echo "Installing Go..."
  rm -fr /usr/local/go /usr/local/bin/go
  curl --silent --show-error --location --fail https://go.dev/dl/go1.19.6.linux-amd64.tar.gz | sudo tar --extract --gzip --file=- --directory=/usr/local
  sudo ln -s /usr/local/go/bin/go /usr/local/bin/go
  go version
}

function install-docker() {
  echo "Installing Docker"
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
  echo "Installing GPU drivers"
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
  echo "Installing healthcheck"
  sudo apt-get -y install --no-install-recommends wget gnupg ca-certificates
  wget -O - https://openresty.org/package/pubkey.gpg | sudo apt-key add -
  echo "deb http://openresty.org/package/ubuntu $(lsb_release -sc) main" \
      | sudo tee /etc/apt/sources.list.d/openresty.list
  sudo apt-get update -y
  sudo apt-get -y install --no-install-recommends openresty
  sudo cp /terraform_node/nginx.conf /usr/local/openresty/nginx/conf/nginx.conf
}

function install-ipfs() {
  echo "Installing IPFS"
  wget "https://dist.ipfs.tech/go-ipfs/${IPFS_VERSION}/go-ipfs_${IPFS_VERSION}_linux-amd64.tar.gz"
  tar -xvzf "go-ipfs_${IPFS_VERSION}_linux-amd64.tar.gz"
  # TODO should reset PWD to home dir after each function call
  cd go-ipfs
  sudo bash install.sh
  ipfs --version
}

function install-bacalhau() {
  if [[ -n "${BACALHAU_BRANCH}" ]] ; then
    install-bacalhau-from-source
  elif [[ -n "${BACALHAU_VERSION}" ]] ; then
    install-bacalhau-from-release
  else
    echo "No bacalhau version or branch specified. Not installing bacalhau."
    exit 1
  fi
}

function install-bacalhau-from-release() {
  echo "Installing Bacalhau from release ${BACALHAU_VERSION}"
  sudo apt-get -y install --no-install-recommends jq
  wget "https://github.com/filecoin-project/bacalhau/releases/download/${BACALHAU_VERSION}/bacalhau_${BACALHAU_VERSION}_linux_amd64.tar.gz"
  tar xfv "bacalhau_${BACALHAU_VERSION}_linux_amd64.tar.gz"
  sudo mv ./bacalhau /usr/local/bin/bacalhau
}

function install-bacalhau-from-source() {
  echo "Installing Bacalhau from branch ${BACALHAU_BRANCH}"
  sudo apt-get -y install --no-install-recommends jq
  git clone --depth 1 --branch ${BACALHAU_BRANCH} https://github.com/filecoin-project/bacalhau.git
  cd bacalhau
  GO111MODULE=on CGO_ENABLED=0 go build -gcflags '-N -l' -trimpath -o ./bacalhau
  sudo mv ./bacalhau /usr/local/bin/bacalhau
}

function install-otel-collector() {
  echo "Installing otel collector"
  if [[ -z "${OTEL_COLLECTOR_VERSION}" ]] ; then
    echo 'OTEL_COLLECTOR_VERSION is undefined. Skipping otel collector installation.'
  else
    sudo apt -y update
    sudo groupadd --system otel
    sudo useradd -s /sbin/nologin --system -g otel otel
    sudo mkdir -p /etc/otel
    sudo mkdir -p /var/lib/otel
    wget "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v${OTEL_COLLECTOR_VERSION}/otelcol-contrib_${OTEL_COLLECTOR_VERSION}_linux_amd64.tar.gz"
    tar xvf "otelcol-contrib_${OTEL_COLLECTOR_VERSION}_linux_amd64.tar.gz"
    sudo mv otelcol-contrib /usr/local/bin/otelcol
    # config file
    sudo tee /terraform_node/otel-collector.yml > /dev/null <<EOF

extensions:
  health_check:
  zpages:
    endpoint: :55679
  basicauth/prometheus:
    client_auth:
      username: ${GRAFANA_CLOUD_PROMETHEUS_USER}
      password: ${GRAFANA_CLOUD_PROMETHEUS_API_KEY}
  basicauth/tempo:
    client_auth:
      username: ${GRAFANA_CLOUD_TEMPO_USER}
      password: ${GRAFANA_CLOUD_TEMPO_API_KEY}
  basicauth/loki:
    client_auth:
      username: ${GRAFANA_CLOUD_LOKI_USER}
      password: ${GRAFANA_CLOUD_LOKI_API_KEY}

receivers:
  hostmetrics:
    scrapers:
      cpu:
      disk:
      load:
      filesystem:
      memory:
      network:
      paging:
  otlp:
    protocols:
      http:
  prometheus:
    config:
      scrape_configs:
        - job_name: 'otel-collector'
          scrape_interval: 5s
          static_configs:
            - targets: [ '0.0.0.0:8888' ]

exporters:
  prometheusremotewrite:
    endpoint: ${GRAFANA_CLOUD_PROMETHEUS_ENDPOINT}
    auth:
      authenticator: basicauth/prometheus
    resource_to_telemetry_conversion:
      enabled: true
  otlp:
    endpoint: ${GRAFANA_CLOUD_TEMPO_ENDPOINT}
    auth:
      authenticator: basicauth/tempo
  loki:
    endpoint: https://${GRAFANA_CLOUD_LOKI_ENDPOINT}/loki/api/v1/push
    auth:
      authenticator: basicauth/loki

processors:
  batch:
  memory_limiter:
    check_interval: 5s
    limit_mib: 4000
    spike_limit_mib: 500
  resourcedetection/gcp:
    detectors: [ env, gcp ]
    timeout: 2s
    override: false
  resource:
    attributes:
    - key: deployment.environment
      value: ${TERRAFORM_WORKSPACE}
      action: insert
    - key: service.namespace
      value: bacalhau
      action: insert
  attributes/metrics:
    actions:
    - pattern: net\.sock.+
      action: delete

service:
  extensions: [basicauth/tempo, basicauth/prometheus, basicauth/loki, zpages, health_check]
  pipelines:
EOF

    if [[ -n "${GRAFANA_CLOUD_PROMETHEUS_ENDPOINT}" ]] && [[ -n "${GRAFANA_CLOUD_PROMETHEUS_USER}" ]] && [[ -n "${GRAFANA_CLOUD_PROMETHEUS_API_KEY}" ]]; then
      sudo tee -a /terraform_node/otel-collector.yml > /dev/null <<EOF
    traces:
      receivers: [otlp]
      processors: [memory_limiter, resourcedetection/gcp, resource, batch]
      exporters: [otlp]
EOF
    fi

    if [[ -n "${GRAFANA_CLOUD_TEMPO_ENDPOINT}" ]] && [[ -n "${GRAFANA_CLOUD_TEMPO_USER}" ]] && [[ -n "${GRAFANA_CLOUD_TEMPO_API_KEY}" ]]; then
      sudo tee -a /terraform_node/otel-collector.yml > /dev/null <<EOF
    metrics:
      receivers: [otlp, prometheus, hostmetrics]
      processors: [memory_limiter, resourcedetection/gcp, resource, attributes/metrics, batch]
      exporters: [prometheusremotewrite]
EOF
    fi

    if [[ -n "${GRAFANA_CLOUD_LOKI_ENDPOINT}" ]] && [[ -n "${GRAFANA_CLOUD_LOKI_USER}" ]] && [[ -n "${GRAFANA_CLOUD_LOKI_API_KEY}" ]]; then
      sudo tee -a /terraform_node/otel-collector.yml > /dev/null <<EOF

# disabled until promtail receiver is merged in collector-contrib
#    logs:
#      receivers: []
#      processors: [memory_limiter, resourcedetection/gcp, resource, batch]
#      exporters: [loki]
EOF
    fi
    sudo chown -R otel:otel /terraform_node/otel-collector.yml
  fi
}

function install-promtail() {
  echo "Installing Promtail/Loki"
  if [[ -z "${LOKI_VERSION}" ]] || [[ -z "${GRAFANA_CLOUD_LOKI_API_KEY}" ]] || [[ -z "${GRAFANA_CLOUD_LOKI_USER}" ]] || [[ -z "${GRAFANA_CLOUD_LOKI_ENDPOINT}" ]]; then
    echo 'Any of LOKI_VERSION, GRAFANA_CLOUD_LOKI_API_KEY, GRAFANA_CLOUD_LOKI_USER, GRAFANA_CLOUD_LOKI_ENDPOINT env variables is undefined. Skipping Promtail/Loki installation.'
  else
    cd ~
    curl -O -L "https://github.com/grafana/loki/releases/download/v${LOKI_VERSION}/promtail-linux-amd64.zip"
    gunzip -S ".zip" promtail-linux-amd64.zip
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
  - url: https://${GRAFANA_CLOUD_LOKI_USER}:${GRAFANA_CLOUD_LOKI_API_KEY}@${GRAFANA_CLOUD_LOKI_ENDPOINT}/loki/api/v1/push

scrape_configs:
  - job_name: journal
    pipeline_stages:
      - json:
          expressions:
           level:
           msg:
      - drop:
          source: "level"
          expression:  "(debug|trace)"
    journal:
      max_age: 12h
      labels:
        job: systemd-journal
        host: ${HOSTNAME}
        label_project: bacalhau
        environment: ${TERRAFORM_WORKSPACE}
    relabel_configs:
      - action: keep
        source_labels: [__journal__systemd_unit]
        regex: '^bacalhau\.service$'
      - source_labels: ['__journal__systemd_unit']
        target_label: 'systemd_unit'
EOF
    sudo mkdir -p /etc/promtail
    sudo cp /terraform_node/promtail.yml /etc/promtail/config.yml
  fi
}

function mount-disk() { 
  echo "Mounting disk"
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
  echo "Initializing IPFS"
  sudo mkdir -p /data/ipfs
  export IPFS_PATH=/data/ipfs

  if [[ ! -e /data/ipfs/version ]]; then
    ipfs init
  fi
}

# install any secrets provided as terraform vars
function install-secrets() {
  echo "Installing secrets"
  # set defaults
  export GRAFANA_CLOUD_PROMETHEUS_API_KEY=""
  export GRAFANA_CLOUD_TEMPO_API_KEY=""
  export GRAFANA_CLOUD_LOKI_API_KEY=""
  export ESTUARY_API_KEY=""
  if [[ -e /data/secrets.sh ]]; then
    source /data/secrets.sh
  fi

  # load new values if they were provided
  if [[ -n "${SECRETS_GRAFANA_CLOUD_PROMETHEUS_API_KEY}" ]]; then
    export GRAFANA_CLOUD_PROMETHEUS_API_KEY="${SECRETS_GRAFANA_CLOUD_PROMETHEUS_API_KEY}"
  fi
  if [[ -n "${SECRETS_GRAFANA_CLOUD_TEMPO_API_KEY}" ]]; then
      export GRAFANA_CLOUD_TEMPO_API_KEY="${SECRETS_GRAFANA_CLOUD_TEMPO_API_KEY}"
  fi
  if [[ -n "${SECRETS_GRAFANA_CLOUD_LOKI_API_KEY}" ]]; then
      export GRAFANA_CLOUD_LOKI_API_KEY="${SECRETS_GRAFANA_CLOUD_LOKI_API_KEY}"
  fi
  if [[ -n "${SECRETS_ESTUARY_API_KEY}" ]]; then
    export ESTUARY_API_KEY="${SECRETS_ESTUARY_API_KEY}"
  fi

  # write the secrets to persistent disk
  sudo tee /data/secrets.sh > /dev/null <<EOG
export GRAFANA_CLOUD_PROMETHEUS_API_KEY="${GRAFANA_CLOUD_PROMETHEUS_API_KEY}"
export GRAFANA_CLOUD_TEMPO_API_KEY="${GRAFANA_CLOUD_TEMPO_API_KEY}"
export GRAFANA_CLOUD_LOKI_API_KEY="${GRAFANA_CLOUD_LOKI_API_KEY}"
export ESTUARY_API_KEY="${ESTUARY_API_KEY}"
EOG

  # clean up variables file from any secret
  sed -e '/^export SECRETS_/d' /terraform_node/variables | sudo tee /terraform_node/variables > /dev/null
}

# if we are node zero, are in unsafe mode and don't have a private key
# then let's copy the unsafe private key so we have a deterministic id
# that other nodes will connect to
function init-bacalhau() {
  echo "Initializing Bacalhau"
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
  sudo systemctl enable ipfs
  sudo systemctl enable bacalhau
  sudo systemctl enable otel
  sudo systemctl enable promtail
  sudo systemctl start ipfs
  sudo systemctl start bacalhau
  sudo systemctl start otel
  sudo systemctl start promtail
  sudo service openresty reload
}

function install() {
  install-go
  install-docker
  install-gpu
  install-healthcheck
  install-ipfs
  install-bacalhau
  mount-disk
  init-ipfs
  init-bacalhau
  install-secrets
  install-otel-collector
  install-promtail
  start-services
}

install
