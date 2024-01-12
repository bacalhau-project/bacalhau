output "requester_public_ip" {
  value = google_compute_instance.requester.*.network_interface.0.access_config.0.nat_ip
}

output "compute_public_ip" {
  value = google_compute_instance.compute.*.network_interface.0.access_config.0.nat_ip
}
