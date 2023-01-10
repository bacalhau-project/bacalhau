#!/bin/sh
set -eux

# Write out our supplied config to disk.
mkdir -p /etc/bacalhau
echo $BACALHAU_HTTP_CLIENTS > /etc/bacalhau/allowed-clients.txt
echo $BACALHAU_HTTP_DOMAINS > /etc/bacalhau/allowed-domains.txt

# Don't forward any packets... otherwise our proxy can be bypassed.
iptables -P FORWARD DROP

# Only accept packets for our HTTP proxy from our internal subnet,
# or for connections we initiated, or internal packets.
iptables -P INPUT DROP
iptables -A INPUT -i lo -j ACCEPT
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
for BRIDGE_SUBNET in $(cat /etc/bacalhau/allowed-clients.txt); do
    iptables -A INPUT -p tcp --src $BRIDGE_SUBNET --dport 8080 -j ACCEPT
done

# Apply rate limits to the outbound connections. We just do this for all
# interfaces rather than working out which is our Internet connection.
for IFACE in $(ip --json address show | jq -rc '.[] | .ifname'); do
    tc qdisc add dev $IFACE root tbf rate 1mbit burst 32kbit latency 10sec
done

# Add Bacalhau job ID to outgoing requests. We can use this to detect jobs
# trying to spawn other jobs.
echo request_header_access X-Bacalhau-Job-ID deny all > /etc/squid/conf.d/bac-job.conf
echo request_header_add X-Bacalhau-Job-ID "$BACALHAU_JOB_ID" all >> /etc/squid/conf.d/bac-job.conf

# Now that everything is configured, run Squid.
squid -d2
sleep 1
tail -f /var/log/squid/access.log
