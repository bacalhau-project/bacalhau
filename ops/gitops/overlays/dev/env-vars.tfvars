bacalhau_version       = "v0.1.43"
bacalhau_port          = "1235"
bacalhau_connect_node0 = "QmNXczFhX8oLEeuGThGowkcJDJUnX4HqoYQ2uaYhuCNSxD"
ipfs_version           = "v0.12.2"
gcp_project            = "bacalhau-development"
instance_count         = 2
region                 = "europe-north1"
zone                   = "europe-north1-c"
volume_size_gb         = 10
machine_type           = "e2-standard-4"
protect_resources      = true
auto_subnets           = false
ingress_cidrs          = ["0.0.0.0/0"]
ssh_access_cidrs       = ["0.0.0.0/0"]
num_gpu_machines       = 0
internal_ip_addresses  = ["0.0.0.0/0","0.0.0.0/0","0.0.0.0/0"]
