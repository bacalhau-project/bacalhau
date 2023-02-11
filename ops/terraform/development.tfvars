bacalhau_version       = ""
bacalhau_branch        = "main"
bacalhau_port          = "1235"
bacalhau_node_id_0     = "QmNXczFhX8oLEeuGThGowkcJDJUnX4HqoYQ2uaYhuCNSxD"
bacalhau_node_id_1     = "QmfRDVYnEcPassyJFGQw8Wt4t9QuA843uuKPVNEVNm4Smo"
ipfs_version           = "v0.12.2"
gcp_project            = "bacalhau-development"
grafana_cloud_prometheus_user      = "14299"
grafana_cloud_prometheus_endpoint  = "https://prometheus-us-central1.grafana.net/api/prom/push"
loki_version                = "2.7.1"
grafana_cloud_loki_user     = "6143"
grafana_cloud_loki_endpoint = "logs-prod-us-central1.grafana.net"
grafana_cloud_tempo_user    = "374169"
grafana_cloud_tempo_endpoint = "tempo-us-central1.grafana.net:443"
instance_count         = 2
region                 = "europe-north1"
zone                   = "europe-north1-c"
volume_size_gb         = 10
machine_type           = "e2-standard-4"
protect_resources      = true
auto_subnets           = false
ingress_cidrs          = ["0.0.0.0/0"]
ssh_access_cidrs       = ["0.0.0.0/0"]
internal_ip_addresses  = ["192.168.0.5", "192.168.0.6"]
public_ip_addresses    = ["34.88.147.110", "34.88.135.65"]
num_gpu_machines       = 0
log_level              = "debug"
otel_collector_version  = "0.70.0"
otel_collector_endpoint = "http://localhost:4318"