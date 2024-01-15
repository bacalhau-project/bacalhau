output "requester_public_ips" {
  value = google_compute_instance.requester.*.network_interface.0.access_config.0.nat_ip
}

output "requester_private_ips" {
  value = google_compute_instance.requester.*.network_interface.0.network_ip
}
