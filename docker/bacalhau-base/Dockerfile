FROM ubuntu:24.04

# Install essential tools and certificates
RUN apt-get update && apt-get install -y \
    ca-certificates=20240203 --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

# Automatically set by Docker to be the --platform flag
ARG TARGETPLATFORM

# Take advantage of the format for $TARGETPLATFORM being "OS/ARCH"
# which matches our output directory structure in ./bin
ADD bin/$TARGETPLATFORM/bacalhau /usr/local/bin/bacalhau

ENTRYPOINT ["bacalhau"]

LABEL org.opencontainers.image.source="https://github.com/bacalhau-project/bacalhau"
LABEL org.opencontainers.image.title="Bacalhau"
LABEL org.opencontainers.image.description="The Bacalhau network provides decentralized compute for compute over data. See https://bacalhau.org for more info."
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.url="https://bacalhau.org"