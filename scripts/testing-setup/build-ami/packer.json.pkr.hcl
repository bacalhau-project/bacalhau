variable "ami_name" {
  type    = string
  default = "ami-bacalhau-latest"
}

variable "base_ami" {
  type    = string
  default = "ami-1853ac65"
}

variable "ssh_username" {
  type    = string
  default = "ubuntu"
}

# source blocks are generated from your builders; a source can be referenced in
# build blocks. A build block runs provisioner and post-processors on a
# source. Read the documentation for source blocks here:
# https://www.packer.io/docs/templates/hcl_templates/blocks/source
source "amazon-ebs" "builder_ebs_image" {
  ami_name                = format("%s-%s-%s", "bacalhau_latest_", regex_replace(timestamp(), "[- TZ:]", ""), uuidv4())
  instance_type           = "c6a.xlarge"
  region                  = "eu-west-1"
  shared_credentials_file = "~/.aws/credentials"
  source_ami              = "ami-01061ffae6ddd2896"
  force_deregister        = "true"
  force_delete_snapshot   = "true"
  ssh_pty                 = "true"
  ssh_timeout             = "20m"
  ssh_username            = "ubuntu"

  tags = {
    BuiltBy = "packer"
    Name    = format("%s-%s-%s", "bacalhau-latest", regex_replace(timestamp(), "[- TZ:]", ""), uuidv4())
  }
}

# a build block invokes sources and runs provisioning steps on them. The
# documentation for build blocks can be found here:
# https://www.packer.io/docs/templates/hcl_templates/blocks/build
build {
  description = "Bacalhau image"

  sources = ["source.amazon-ebs.builder_ebs_image"]

  provisioner "shell" {
    inline = [ "cloud-init status --wait"]
  }

  provisioner "shell" {
    inline = [<<EOM
  mkdir -p /tmp/services
  mkdir -p /tmp/scripts
  EOM
    ]
  }

  provisioner "file" {
    destination = "/tmp/services"
    source      = "services/"
  }

  provisioner "file" {
    destination = "/tmp/scripts"
    source      = "scripts/"
  }

  provisioner "shell" {
    inline = [<<EOM
  sudo cp -R /tmp/services/* /etc/systemd/system
  sudo cp -R /tmp/scripts /usr/local/bin/ 
  sudo chmod +x /usr/local/bin/scripts/*
  mkdir -p /home/ubuntu/health_check
  touch /home/ubuntu/health_check/peer_token.html
  usermod -aG docker ubuntu &> /dev/null
  EOM
    ]
  }

  provisioner "shell" {
    expect_disconnect = true
    inline            = ["sudo /usr/local/bin/scripts/update_packages.sh"]

  }

  provisioner "shell" {
    inline = ["sudo /usr/local/bin/scripts/setup_node.sh"]
  }
}
