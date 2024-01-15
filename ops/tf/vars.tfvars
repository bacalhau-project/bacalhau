//Usage: install-bacalhau [ | release <version> | branch <branch-name>]" >&2
// if no args are provided bacalhau will use https://get.bacalhau.org/install.sh
install_bacalhau_argument = "branch frrist/tf"

// there is only 1 requester
requester_machine_type ="e2-standard-8"

// have all the compute you'd like, they will bootstrap to requester automatically
compute_machine_type = "e2-standard-8"
compute_count = 2

gcp_boot_image = "ubuntu-os-cloud/ubuntu-2204-lts"
gcp_project_id = "forrest-dev-407420"
gcp_region = "us-west1"
gcp_zone = "us-west1-b"
