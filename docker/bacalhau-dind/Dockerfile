FROM docker:dind

# Install necessary packages
RUN apk update && apk add --no-cache \
    curl \
    bash \
    coreutils

# Automatically set by Docker to be the --platform flag
ARG TARGETPLATFORM

# Take advantage of the format for $TARGETPLATFORM being "OS/ARCH"
# which matches our output directory structure in ./bin
ADD bin/$TARGETPLATFORM/bacalhau /usr/local/bin/bacalhau

# Add our custom entrypoint script
COPY docker/bacalhau-dind/entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh", "bacalhau"]

LABEL org.opencontainers.image.source="https://github.com/bacalhau-project/bacalhau"
LABEL org.opencontainers.image.title="Bacalhau"
LABEL org.opencontainers.image.description="The Bacalhau network provides distributed compute over data. See https://bacalhau.org for more info."
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.url="https://bacalhau.org"