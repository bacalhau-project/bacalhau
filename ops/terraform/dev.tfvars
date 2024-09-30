bacalhau_version       = "v1.5.0-alpha8"
bacalhau_branch        = ""
bacalhau_port          = "1235"
bacalhau_environment   = "development"
ipfs_version           = "v0.18.1"
gcp_project            = "bacalhau-dev"
grafana_cloud_prometheus_user      = "14299"
grafana_cloud_prometheus_endpoint  = "https://prometheus-us-central1.grafana.net/api/prom/push"
loki_version                = "2.7.1"
grafana_cloud_loki_user     = "6143"
grafana_cloud_loki_endpoint = "logs-prod-us-central1.grafana.net"
grafana_cloud_tempo_user    = "76"
grafana_cloud_tempo_endpoint = "tempo-us-central1.grafana.net:443"
instance_count         = 1
region                 = "us-east4"
zone                   = "us-east4-c"
volume_size_gb         = 100
machine_type           = "e2-standard-2"
protect_resources      = true
auto_subnets           = false
ingress_cidrs          = ["0.0.0.0/0"]
egress_cidrs           = ["0.0.0.0/0"]
ssh_access_cidrs       = ["0.0.0.0/0"]
internal_ip_addresses  = ["192.168.0.5"]
public_ip_addresses    = ["34.86.177.175"]
num_gpu_machines       = 0
log_level              = "debug"
otel_collector_version  = "0.70.0"
otel_collector_endpoint = "http://localhost:4318"
web_ui_enabled          = true
