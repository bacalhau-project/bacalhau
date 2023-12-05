#!/bin/bash
THIS_USER="yourusername"
THIS_IP="0.0.0.0"

apt install -y sudo build-essential

useradd "${THIS_USER}"
usermod -a -G sudo "${THIS_USER}"
sudo su - "${THIS_USER}"
ssh-keygen

sudo more /etc/sudoers # Just a test to make sure sudoers works

# Logout
# trunk-ignore(shellcheck/SC2029)
ssh root@"${THIS_IP}" "cat >> /home/${THIS_USER}/.ssh/authorized_keys" < ~/.ssh/id_rsa.pub

# Install Docker
sudo mkdir -p /etc/apt/keyrings
# trunk-ignore(shellcheck/SC2312)
echo   "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  # trunk-ignore(shellcheck/SC2312)
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo usermod -aG docker "${USER}"
docker run hello-world

# Install brew
sudo apt-get update
# trunk-ignore(shellcheck/SC2312)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install go
brew install go gcc@5 gcc make hyperfine
gh repo clone bacalhau-project/bacalhau
