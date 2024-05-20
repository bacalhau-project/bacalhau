requester_machine_type ="e2-standard-8"

// have all the compute you'd like, they will bootstrap to requester automatically
compute_machine_type = "e2-standard-8"
compute_count = 1

gcp_boot_image = "projects/bacalhau-dev/global/images/bacalhau-v1-3-0-ubuntu-2204-lts"
gcp_project_id = "forrest-dev-407420"
gcp_region = "us-west1"
gcp_zone = "us-west1-b"

// HMAC keys for accessing buckets over S3 protocol (GCP Bucket or AWS S3 Bucket)
aws_access_key_id = ""
aws_secret_access_key = ""

bacalhau_accept_networked_jobs = true

bacalhau_otel_collector_endpoint = "http://analytics.bacalhau.tech:4318"

bacalhau_requester_api_token = "password"
bacalhau_compute_api_token = "password"

bacalhau_install_branch = ""
bacalhau_install_version = ""
bacalhau_install_commit = ""
