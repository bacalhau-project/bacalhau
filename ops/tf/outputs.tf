output "requester_public_ip" {
  value = module.requester_instance.requester_public_ips
}

output "compute_public_ip" {
  value = module.compute_instance.compute_public_ips
}

output "bacalhau_secret_auth_token" {
  value = random_string.bacalhau_auth_token.result
}