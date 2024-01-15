output "vpc_network_name" {
  value = google_compute_network.vpc_network.name
}

output "subnetwork_name" {
  value = google_compute_subnetwork.subnetwork.name
}

output "requester_ip" {
  value = google_compute_address.requester_ip.address
}