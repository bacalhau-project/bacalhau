output "compute_private_ips" {
  value = [for instance in google_compute_instance.compute : instance.network_interface[0].network_ip]
}

output "compute_public_ips" {
  value = google_compute_instance.compute.*.network_interface.0.access_config.0.nat_ip
}

