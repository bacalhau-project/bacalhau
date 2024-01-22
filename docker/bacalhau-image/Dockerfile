# syntax=docker/dockerfile:1.4
FROM cgr.dev/chainguard/nvidia-device-plugin

# Automatically set by Docker to be the --platform flag
ARG TARGETPLATFORM

# Take advantage of the format for $TARGETPLATFORM being "OS/ARCH"
# which matches our output directory structure in ./bin
ADD bin/$TARGETPLATFORM/bacalhau /usr/local/bin/bacalhau
ENV PATH="/usr/local/bin:/usr/bin"
ENTRYPOINT ["bacalhau"]
LABEL org.opencontainers.image.source https://github.com/bacalhau-project/bacalhau
LABEL org.opencontainers.image.title "Bacalhau"
LABEL org.opencontainers.image.description "The Bacalhau network provides decentralized compute for compute over data. See https://bacalhau.org for more info."
LABEL org.opencontainers.image.licenses Apache-2.0
LABEL org.opencontainers.image.url https://bacalhau.org
