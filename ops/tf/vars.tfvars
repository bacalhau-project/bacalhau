//Usage: install-bacalhau [ | release <version> | branch <branch-name>]" >&2
// if no args are provided bacalhau will use https://get.bacalhau.org/install.sh
// there is only 1 requester
requester_machine_type ="e2-standard-8"

// have all the compute you'd like, they will bootstrap to requester automatically
compute_machine_type = "n1-standard-8"
compute_count = 2

gcp_boot_image_requester = "projects/forrest-dev-407420/global/images/bacalhau-ubuntu-2004-lts-test"
# Comment this line
gcp_boot_image_compute = "projects/forrest-dev-407420/global/images/bacalhau-ubuntu-2004-lts-test"
# And Uncomment this one to use GPUs
# gcp_boot_image_compute = "projects/forrest-dev-407420/global/images/bacalhau-common-cu121-v20230925-ubuntu-2004-lts-test-16"

gcp_project_id = "forrest-dev-407420"
gcp_region = "us-west1"
gcp_zone = "us-west1-b"

// HMAC keys for accessing buckets over S3 protocol (GCP Bucket or AWS S3 Bucket)
aws_access_key_id = ""
aws_secret_access_key = ""

bacalhau_accept_networked_jobs = true
accelerator = "nvidia-tesla-t4"
# Make this zero for no GPUs, Increase the count to one or more to attach GPUs
accelerator_count = 0