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
          expression:  "(trace)"
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
    export AWS_ACCESS_KEY_ID=""
    export AWS_SECRET_ACCESS_KEY=""
    export DOCKER_USERNAME=""
    export DOCKER_PASSWORD=""

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
    if [[ -n "${SECRETS_AWS_ACCESS_KEY_ID}" ]]; then
        export AWS_ACCESS_KEY_ID="${SECRETS_AWS_ACCESS_KEY_ID}"
    fi
    if [[ -n "${SECRETS_AWS_SECRET_ACCESS_KEY}" ]]; then
        export AWS_SECRET_ACCESS_KEY="${SECRETS_AWS_SECRET_ACCESS_KEY}"
    fi
    if [[ -n "${SECRETS_DOCKER_USERNAME}" ]]; then
        export DOCKER_USERNAME="${SECRETS_DOCKER_USERNAME}"
    fi
    if [[ -n "${SECRETS_DOCKER_PASSWORD}" ]]; then
        export DOCKER_PASSWORD="${SECRETS_DOCKER_PASSWORD}"
    fi

    # write the secrets to persistent disk
  sudo tee /data/secrets.sh > /dev/null <<EOG
export GRAFANA_CLOUD_PROMETHEUS_API_KEY="${GRAFANA_CLOUD_PROMETHEUS_API_KEY}"
export GRAFANA_CLOUD_TEMPO_API_KEY="${GRAFANA_CLOUD_TEMPO_API_KEY}"
export GRAFANA_CLOUD_LOKI_API_KEY="${GRAFANA_CLOUD_LOKI_API_KEY}"
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY}"
export DOCKER_USERNAME="${DOCKER_USERNAME}"
export DOCKER_PASSWORD="${DOCKER_PASSWORD}"
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
    sudo service openresty reload
    sudo systemctl start ipfs
    sudo systemctl start bacalhau
    sudo systemctl start otel
    sudo systemctl start promtail
}

function install() {
    install-go
    install-docker
    install-git
    install-git-lfs
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
