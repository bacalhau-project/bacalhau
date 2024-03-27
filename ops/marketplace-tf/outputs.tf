output "requester_public_ip" {
  value = module.requester_instance.requester_public_ips
}

output "compute_public_ip" {
  value = module.compute_instance.compute_public_ips
}

output "bacalhau_requester_api_token" {
  value = local.token_config.requester_api_token
}

output "bacalhau_compute_api_token" {
  value = local.token_config.compute_api_token
}

output "tls_cert" {
  value = tls_self_signed_cert.tlscert.cert_pem
}
