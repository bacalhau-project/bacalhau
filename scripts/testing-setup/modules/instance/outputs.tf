output "instance_public_ip" {
  description = "Public IP address of the EC2 instance"
  value       = aws_instance.bacalhau_node.public_ip
}


output "instance_private_ip" {
  description = "Private IP address of the EC2 instance"
  value       = aws_instance.bacalhau_node.private_ip
}

output "public_dns" {
  description = "Public DNS"
  value       = aws_instance.bacalhau_node.public_dns
}

output "private_dns" {
  description = "Private DNS"
  value       = aws_instance.bacalhau_node.private_dns
}
