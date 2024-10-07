bacalhau_version            = "v1.5.0-alpha14"
bacalhau_branch             = "" # deploy from a branch instead of the version above
bacalhau_port               = "1235"
bacalhau_environment        = "staging"
ipfs_version                = "v0.18.1"
gcp_project                 = "bacalhau-stage"
grafana_cloud_prometheus_user      = "1008771"
grafana_cloud_prometheus_endpoint  = "https://prometheus-us-central1.grafana.net/api/prom/push"
loki_version                = "2.7.1"
grafana_cloud_loki_user     = "606991"
grafana_cloud_loki_endpoint = "logs-prod-017.grafana.net"
grafana_cloud_tempo_user    = "603503"
grafana_cloud_tempo_endpoint = "tempo-us-central1.grafana.net:443"
instance_count              = 2
region                      = "us-east4"
zone                        = "us-east4-c"
volume_size_gb              = 100
boot_disk_size_gb           = 100
machine_type                = "e2-standard-2"
protect_resources           = true
auto_subnets                = true
ingress_cidrs               = ["0.0.0.0/0"]
egress_cidrs                = ["0.0.0.0/0"]
ssh_access_cidrs            = ["0.0.0.0/0"]
num_gpu_machines            = 0
internal_ip_addresses       = ["10.150.0.5", "10.150.0.6"]
public_ip_addresses         = ["34.85.228.65", "34.86.73.105"]
log_level                   = "debug"
otel_collector_version  = "0.70.0"
otel_collector_endpoint = "http://localhost:4318"
web_ui_enabled          = true