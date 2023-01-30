# Dockerfile for Bacalhau HTTP gateway
#
# This Dockerfile sets up a container image containing an HTTP(S) proxy with a
# specific allow-list of Internet domains that should be accessible. The proxy
# allows access to these domains only and blocks all other traffic. The
# container also enforces rate limits on the traffic.
#
# The container is designed to be attached to two networks:
#
# 1. A bridge connecting it to all containers taking part in the job, which is
#    internal only and not connected to the Internet
# 2. A host network that is Internet connected, which only the proxy can access
#
# This enforces that the containers in the bridge network can only access the
# external network via the proxy.
#
# The image uses Squid as an HTTP(S) proxy, iptables to filter packet flows and
# iproute2 to do traffic shaping. See the squid.conf for how the allow-lists are
# managed and gateway.sh for how the traffic control is configured.
#
# In particular, the image expects some environment variables to be supplied:
#
# - BACALHAU_HTTP_CLIENTS which is a JSON array of strings of subnets allowed to
#   access the gateway
# - BACALHAU_HTTP_DOMAINS which is a JSON array of strings of domains that
#   clients are allowed to access
# - BACALHAU_JOB_ID which contains the ID of the Bacalhau job being run
#
# The container needs to be started with --cap-add=NET_ADMIN so that it can
# configure iptables and traffic control.

FROM ubuntu:22.04
RUN apt update && apt install -y squid iptables iproute2 jq curl \
    && rm -rf /var/lib/apt/lists/*

COPY squid.conf /etc/squid/conf.d/
COPY gateway.sh /usr/local/bin
COPY health_check.sh /usr/local/bin

CMD ["bash", "/usr/local/bin/gateway.sh"]
HEALTHCHECK --interval=1s --start-period=5s CMD ["bash", "/usr/local/bin/health_check.sh" ]
