#!/bin/bash
# external_ip=$$(curl -s -H 'Metadata-Flavor: Google' "http://metadata/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip")

# echo "{ 'hostname': '$$(hostname -f)', 'date': '$$(date --rfc-3339=ns)', 'ip': '$$external_ip'}"

echo "OK"