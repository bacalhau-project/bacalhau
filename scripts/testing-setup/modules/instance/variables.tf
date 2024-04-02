variable "PATH_TO_PUBLIC_KEY" {
  description = "Path to public key to push into authorized_keys"
  type        = string
}

variable "PATH_TO_PRIVATE_KEY" {
  description = "Path to private key to login"
  type        = string
}


variable "SUBNET_ID" {
  description = "Id of the subnet"
  type        = string
}

variable "AWS_INTERNET_GATEWAY_ID" {
  description = "AWS Internet Gateway ID"
  type        = string
}

variable "SECURITY_GROUP_ALLOW_SSH_ID" {
  description = "AWS Internet Gateway ID"
  type        = string
}

variable "AWS_KEY_PAIR_DEPLOYER_KEY_NAME" {
  description = "AWS Internet Gateway ID"
  type        = string
}

variable "INSTANCE_TYPE" {
  description = "AWS instance type"
  type        = string
}

variable "AMIS" {
  type = map(string)
}
variable "AWS_REGION" {
  type = string
}

variable "NODE_NUMBER" {
  type = string
}
