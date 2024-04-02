variable "AWS_REGION" {
  default = "eu-west-1"
}

variable "PATH_TO_PRIVATE_KEY" {
  default = "~/.ssh/id_rsa"
}
variable "PATH_TO_PUBLIC_KEY" {
  default = "~/.ssh/id_rsa.pub"
}
variable "AWS_CREDENTIALS_FILE" {
  default = "~/.aws/credentials"
}

variable "AMIS" {
  type = map(string)
  default = {
    eu-west-1 = "ami-05618744ec24af4f8"
  }
}

variable "INSTANCE_USERNAME" {
  default = "ubuntu"
}

variable "NUMBER_OF_NODES" {
  default = 3
}
