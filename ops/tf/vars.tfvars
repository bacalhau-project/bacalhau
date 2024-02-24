//Usage: install-bacalhau [ | release <version> | branch <branch-name>]" >&2
// if no args are provided bacalhau will use https://get.bacalhau.org/install.sh
// there is only 1 requester
requester_machine_type ="e2-standard-8"

// have all the compute you'd like, they will bootstrap to requester automatically
compute_machine_type = "e2-standard-8"
compute_count = 4

gcp_boot_image = "projects/forrest-dev-407420/global/images/bacalhau-ubuntu-2204-lts-test-latest"
gcp_project_id = "bacalhau-video-processing"
gcp_region = "northamerica-northeast2"
gcp_zone = "northamerica-northeast2-a"

// HMAC keys for accessing buckets over S3 protocol (GCP Bucket or AWS S3 Bucket)
aws_access_key_id = ""
aws_secret_access_key = ""

bacalhau_accept_networked_jobs = true

bacalhau_otel_collector_endpoint = "http://analytics.bacalhau.tech:4318"

bacalhau_requester_api_token = "password"
bacalhau_compute_api_token = "password"

bacalhau_install_branch = ""
bacalhau_install_version = ""
bacalhau_install_commit = "fc34f699d1d92889ecd3d365d32a8c3344296fd6"
