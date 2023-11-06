bacalhau_version            = "v1.1.3"
bacalhau_port               = "1235"
bacalhau_node_id_0          = "QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL"
bacalhau_node_id_1          = "QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF"
bacalhau_node_id_2          = "QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3"
ipfs_version                = "v0.12.2"
gcp_project                 = "bacalhau-production"
grafana_cloud_prometheus_user      = "1008771"
grafana_cloud_prometheus_endpoint  = "https://prometheus-us-central1.grafana.net/api/prom/push"
loki_version                = "2.7.1"
grafana_cloud_loki_user     = "606991"
grafana_cloud_loki_endpoint = "logs-prod-017.grafana.net"
grafana_cloud_tempo_user    = "603503"
grafana_cloud_tempo_endpoint = "tempo-us-central1.grafana.net:443"
instance_count              = 5
region                      = "us-east4"
zone                        = "us-east4-c"
# When increasing the volume size, you may need to manually resize the filesystem
# on the virtual machine. If `df -h` shows only 1000, then `sudo resize2fs /dev/sdb` 
# will resize the /data (/dev/sdb) drive to use all of the space. Do not use for 
# boot disk.
volume_size_gb              = 2000 # when increasing this value you need to claim the new space manually
boot_disk_size_gb           = 1000
machine_type                = "e2-standard-16"
protect_resources           = true
auto_subnets                = true
ingress_cidrs               = ["0.0.0.0/0"]
ssh_access_cidrs            = ["0.0.0.0/0"]
num_gpu_machines            = 2
internal_ip_addresses       = ["10.150.0.5", "10.150.0.6", "10.150.0.7", "10.150.0.8", "10.164.0.9"]
public_ip_addresses         = ["35.245.115.191", "35.245.61.251", "35.245.251.239", "34.150.153.87", "34.91.247.176"]
log_level                   = "debug"
otel_collector_version  = "0.70.0"
otel_collector_endpoint = "http://localhost:4318"
