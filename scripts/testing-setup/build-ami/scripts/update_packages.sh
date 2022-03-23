#!/bin/bash

apt update -y
apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    net-tools \
    gnupg
curl -fsSL https://download.docker.com/linux/ubuntu/gpg > /tmp/docker.signature
gpg --dearmor -o --batch --yes /usr/share/keyrings/docker-archive-keyring.gpg /tmp/docker.signature

echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list

apt-get install -y docker-ce docker-ce-cli containerd.io
usermod -aG docker ubuntu