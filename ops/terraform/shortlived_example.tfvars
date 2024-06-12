bacalhau_version        = "v0.1.29"
bacalhau_port           = "1235"
ipfs_version            = "v0.12.2"
gcp_project             = "bacalhau-cicd"
instance_count          = 2
region                  = "europe-north1"
zone                    = "europe-north1-c"
volume_size_gb          = 100
boot_disk_size_gb       = 100
machine_type            = "e2-standard-4"
protect_resources       = false
bacalhau_unsafe_cluster = true
auto_subnets            = false
num_gpu_machines        = 0

# gcloud compute instances create performance-test-$(git rev-parse --short HEAD) --project=bacalhau-cicd --zone=europe-north1-c --machine-type=e2-standard-4 --network-interface=network-tier=PREMIUM,subnet=default --no-restart-on-failure --maintenance-policy=TERMINATE --provisioning-model=SPOT --instance-termination-action=DELETE --service-account=702126710047-compute@developer.gserviceaccount.com --scopes=https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring.write,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/service.management.readonly,https://www.googleapis.com/auth/trace.append --create-disk=auto-delete=yes,boot=yes,device-name=performance-test-$(git rev-parse --short HEAD)-disk,image=projects/ubuntu-os-cloud/global/images/ubuntu-2204-jammy-v20220712a,mode=rw,size=100,type=projects/bacalhau-cicd/zones/europe-north1-c/diskTypes/pd-balanced --no-shielded-secure-boot --shielded-vtpm --shielded-integrity-monitoring --reservation-affinity=any
