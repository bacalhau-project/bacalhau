#!/usr/bin/env bash

# Exit on error. Append || true if you expect an error.
set -o errexit
# Exit on error inside any functions or subshells.
set -o errtrace
# Do not allow use of undefined vars. Use ${VAR:-} to use an undefined VAR
set -o nounset
# Catch the error in case mysqldump fails (but gzip succeeds) in `mysqldump |gzip`
set -o pipefail
# Turn on traces, useful while debugging but commented out by default
#set -o xtrace

# Write out our supplied config to disk.
mkdir -p /etc/bacalhau
echo "${BACALHAU_HTTP_CLIENTS}" | jq -r '.[]' > /etc/bacalhau/allowed-clients.txt
echo "${BACALHAU_HTTP_DOMAINS}" | jq -r '.[]' > /etc/bacalhau/allowed-domains.txt

# Don't forward any packets... otherwise our proxy can be bypassed.
iptables -P FORWARD DROP

# Only accept packets for our HTTP proxy from our internal subnet,
# or for connections we initiated, or internal packets.
iptables -P INPUT DROP
iptables -A INPUT -i lo -j ACCEPT
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

while IFS= read -r BRIDGE_SUBNET; do
    iptables -A INPUT -p tcp --src "${BRIDGE_SUBNET}" --dport 8080 -j ACCEPT
done < <(cat /etc/bacalhau/allowed-clients.txt)

# Apply rate limits to the outbound connections. We just do this for all
# interfaces rather than working out which is our Internet connection.
while IFS= read -r IFACE; do
    tc qdisc add dev "${IFACE}" root tbf rate 10mbit burst 32kbit latency 10sec
done < <(ip --json address show | jq -rc '.[] | .ifname')

# Add Bacalhau job ID to outgoing requests. We can use this to detect jobs
# trying to spawn other jobs.
echo request_header_access X-Bacalhau-Job-ID deny all > /etc/squid/conf.d/bac-job.conf
echo request_header_add X-Bacalhau-Job-ID "${BACALHAU_JOB_ID}" all >> /etc/squid/conf.d/bac-job.conf

# Make sure the access log is present for us to tail at the end, even if squid hasn't logged anything yet
touch /var/log/squid/access.log

# Now that everything is configured, run Squid.
squid -d2
tail -F /var/log/squid/access.log
