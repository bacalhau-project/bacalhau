resource "aws_instance" "bacalhau_node" {
  ami           = lookup(var.AMIS, var.AWS_REGION)
  instance_type = var.INSTANCE_TYPE
  

  # the VPC subnet
  subnet_id = var.SUBNET_ID
  # the security group
  vpc_security_group_ids = ["${var.SECURITY_GROUP_ALLOW_SSH_ID}"]
  # the public SSH key
  key_name = var.AWS_KEY_PAIR_DEPLOYER_KEY_NAME

  tags = {
    Name = "bacalhau_node_${var.NODE_NUMBER}"
  }
}

