#!/bin/bash

apt update
apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    net-tools 2> /dev/null
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg 2> /dev/null
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
apt-get update 2> /dev/null
apt-get -y install docker-ce docker-ce-cli containerd.io 2> /dev/null
usermod -aG docker ubuntu &> /dev/null