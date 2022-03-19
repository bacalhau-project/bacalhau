provider "aws" {
  region                   = var.AWS_REGION
  shared_credentials_files = [var.AWS_CREDENTIALS_FILE]
}

resource "aws_security_group" "allow_ssh_and_bacalhau" {
  vpc_id      = aws_vpc.bacalhau_vpc.id
  name        = "allow_ssh_and_bacalhau"
  description = "security group that allows ssh and bacalhau and all egress traffic"
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
  }


  tags = {
    Name = "allow_ssh_and_bacalhau"
  }
}

# https://geekdudes.wordpress.com/2018/01/09/install-packages-to-amazon-virtual-machine-using-terraform/

# Internet VPC
resource "aws_vpc" "bacalhau_vpc" {
  cidr_block           = "10.0.0.0/16"
  instance_tenancy     = "default"
  enable_dns_support   = "true"
  enable_dns_hostnames = "true"
  enable_classiclink   = "false"
  tags = {
    Name = "bacalhau_vpc"
  }
}


# Subnets
resource "aws_subnet" "bacalhau_public_1" {
  vpc_id                  = aws_vpc.bacalhau_vpc.id
  cidr_block              = "10.0.1.0/24"
  map_public_ip_on_launch = "true"
  availability_zone       = "eu-west-1a"
  tags = {
    Name = "bacalhau_public_1"
  }
}
resource "aws_subnet" "bacalhau_private_1" {
  vpc_id                  = aws_vpc.bacalhau_vpc.id
  cidr_block              = "10.0.2.0/24"
  map_public_ip_on_launch = "false"
  availability_zone       = "eu-west-1a"

  tags = {
    Name = "bacalhau_private_1"
  }
}

# Internet GW
resource "aws_internet_gateway" "bacalhau_gw" {
  vpc_id = aws_vpc.bacalhau_vpc.id

  tags = {
    Name = "bacalhau_vpc_gateway"
  }
}

# route tables
resource "aws_route_table" "bacalhau_public_route_table" {
  vpc_id = aws_vpc.bacalhau_vpc.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.bacalhau_gw.id
  }

  tags = {
    Name = "bacalhau_public_route_table"
  }
}

# route associations public
resource "aws_route_table_association" "bacalhau_public_1_a" {
  subnet_id      = aws_subnet.bacalhau_public_1.id
  route_table_id = aws_route_table.bacalhau_public_route_table.id
}

resource "aws_key_pair" "deployer" {
  key_name   = "deployer-key"
  public_key = file("${var.PATH_TO_PUBLIC_KEY}")
}

module "instance" {
  source = "./modules/instance"

  count = var.NUMBER_OF_NODES

  PATH_TO_PUBLIC_KEY             = var.PATH_TO_PUBLIC_KEY
  PATH_TO_PRIVATE_KEY            = var.PATH_TO_PRIVATE_KEY
  SUBNET_ID                      = aws_subnet.bacalhau_public_1.id
  AWS_INTERNET_GATEWAY_ID        = aws_internet_gateway.bacalhau_gw.id
  SECURITY_GROUP_ALLOW_SSH_ID    = aws_security_group.allow_ssh_and_bacalhau.id
  AWS_KEY_PAIR_DEPLOYER_KEY_NAME = aws_key_pair.deployer.key_name
  AMIS                           = var.AMIS
  AWS_REGION                     = var.AWS_REGION
  INSTANCE_TYPE                  = "t2.micro"
  NODE_NUMBER                    = tostring(count.index)
}

output "instance_public_dns" {
  description = "Public DNS address of the EC2 instance"
  value       = module.instance.*.public_dns
}

output "instance_private_dns" {
  description = "Private DNS address of the EC2 instance"
  value       = module.instance.*.private_dns
}

output "instance_private_ips" {
  description = "Private IPs address of the EC2 instance"
  value       = module.instance.*.instance_private_ip
}
