#!/bin/bash
set -euo pipefail

cd ../terraform/
bash scripts/connect_workspace.sh production

PROD_ENV_VARS=$(gcloud compute ssh bacalhau-vm-production-1 \
    --zone=us-east4-c \
    --command="cat /terraform_node/variables")

PROD_SECRETS=$(gcloud compute ssh bacalhau-vm-production-1 \
    --zone=us-east4-c \
    --command="cat /data/secrets.sh")

for s in 51.81.184.74 51.81.184.112 51.81.184.118 51.81.184.117
do
    ssh ubuntu@${s} sudo mkdir -p /terraform_node
    ssh ubuntu@${s} "echo \"${PROD_ENV_VARS}\" | sudo tee /terraform_node/variables" > /dev/null
    
    # TODO fix this - installed by install-node.sh!!!!
    ssh ubuntu@${s} "echo \"${PROD_SECRETS}\" | sudo tee /data/secrets.sh" > /dev/null
    
    ssh ubuntu@${s} "source /terraform_node/variables; wget -q --no-clobber https://github.com/filecoin-project/bacalhau/archive/refs/tags/\${BACALHAU_VERSION}.tar.gz; mkdir -p bacalhau; tar -xzf \${BACALHAU_VERSION}.tar.gz -C bacalhau --strip-components=1;"
    # At this point we have the bacalhau code located at "~/bacalhau"

    #########
    # node scripts
    #########
    ssh ubuntu@${s} sudo cp ./bacalhau/ops/terraform/remote_files/configs/unsafe-private-key /terraform_node/bacalhau-unsafe-private-key
    ssh ubuntu@${s} sudo cp ./bacalhau/ops/terraform/remote_files/scripts/install-node.sh /terraform_node/install-node.sh
    ssh ubuntu@${s} sudo cp ./bacalhau/ops/terraform/remote_files/scripts/start-bacalhau.sh /terraform_node/start-bacalhau.sh

    #########
    # health checker
    #########
    ssh ubuntu@${s} sudo mkdir -p /var/www/health_checker
    ssh ubuntu@${s} sudo cp ./bacalhau/ops/terraform/remote_files/health_checker/nginx.conf /terraform_node/nginx.conf
    ssh ubuntu@${s} sudo cp ./bacalhau/ops/terraform/remote_files/health_checker/livez.sh /var/www/health_checker/livez.sh
    ssh ubuntu@${s} sudo cp ./bacalhau/ops/terraform/remote_files/health_checker/healthz.sh /var/www/health_checker/healthz.sh
    ssh ubuntu@${s} "echo bacalhau-network-production | sudo tee /var/www/health_checker/network_name.txt" > /dev/null
    ssh ubuntu@${s} "curl https://ipinfo.io/ip | sudo tee /var/www/health_checker/address.txt" > /dev/null
    ssh ubuntu@${s} sudo chmod u+x /var/www/health_checker/*.sh

    #########
    # systemd units
    #########
    ssh ubuntu@${s} sudo cp ./bacalhau/ops/terraform/remote_files/configs/ipfs-daemon.service /etc/systemd/system/ipfs-daemon.service
    ssh ubuntu@${s} sudo cp ./bacalhau/ops/terraform/remote_files/configs/bacalhau-daemon.service /etc/systemd/system/bacalhau-daemon.service
    ssh ubuntu@${s} sudo cp ./bacalhau/ops/terraform/remote_files/configs/prometheus-daemon.service /etc/systemd/system/prometheus-daemon.service

    ##############################
    # run the install script
    ##############################
    
    ssh ubuntu@${s} bash <<EOF
        echo "@@ -194,7 +194,6 @@
   install-healthcheck
   install-ipfs
   install-bacalhau
-  mount-disk
   init-ipfs
   init-bacalhau
   install-secrets" | sudo patch /terraform_node/install-node.sh
EOF

    ssh ubuntu@${s} "sudo sed -i 's/set -euo pipefail/set -eo pipefail/g' /terraform_node/install-node.sh"

    echo "Running install-node.sh"
    ssh -t ubuntu@${s} "sudo bash /terraform_node/install-node.sh 2>&1 | tee -a /tmp/bacalhau.log"

done

