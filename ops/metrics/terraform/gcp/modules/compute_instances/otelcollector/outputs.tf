output "otel_collector_public_ips" {
  value = google_compute_instance.otel_collector.*.network_interface.0.access_config.0.nat_ip
}

output "otel_collector_private_ips" {
  value = google_compute_instance.otel_collector.*.network_interface.0.network_ip
}

