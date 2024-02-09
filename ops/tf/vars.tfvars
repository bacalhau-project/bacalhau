//Usage: install-bacalhau [ | release <version> | branch <branch-name>]" >&2
// if no args are provided bacalhau will use https://get.bacalhau.org/install.sh
// there is only 1 requester
requester_machine_type ="e2-standard-8"

// have all the compute you'd like, they will bootstrap to requester automatically
compute_machine_type = "e2-standard-8"
compute_count = 2

gcp_boot_image = "projects/forrest-dev-407420/global/images/bacalhau-ubuntu-2204-lts-test-latest"
gcp_project_id = "forrest-dev-407420"
gcp_region = "us-west1"
gcp_zone = "us-west1-b"

// HMAC keys for accessing buckets over S3 protocol (GCP Bucket or AWS S3 Bucket)
aws_access_key_id = ""
aws_secret_access_key = ""

bacalhau_accept_networked_jobs = true

bacalhau_otel_collector_endpoint = "http://analytics.bacalhau.tech:4318"

bacalhau_requester_api_token = "token_for_requester_api"
bacalhau_compute_api_token = "token_for_compute_api"

bacalhau_install_branch = ""
bacalhau_install_version = ""
bacalhau_install_commit = "292663a6a23fe9a88dabb2b3bdf797dfa825bbe5"
