#!/bin/bash

# Setup nginx
echo -n "Starting nginx service... "
systemctl daemon-reload
systemctl enable nginx.service
systemctl start  nginx.service
echo "Done."

# Setup bacalhau
echo -n "Downloading bacalhau... "
curl -s https://bacalhau.org/install.sh | bash -- &> /dev/null 
echo "Done."

echo -n "Starting bacalhau service... "
systemctl daemon-reload
systemctl enable bacalhau.service
systemctl start  bacalhau.service
echo "Done."

crontab /home/ubuntu/crontab_entries