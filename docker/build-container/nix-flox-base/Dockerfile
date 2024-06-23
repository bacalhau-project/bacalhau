FROM alpine:latest

# Install required packages
RUN apk add --no-cache sudo shadow bash curl xz git openrc git-lfs nodejs aws-cli just

RUN curl -sSL https://sdk.cloud.google.com | bash

# Install Nix
ENV NIX_INSTALLER_EXTRA_CONF='filter-syscalls = false'
ENV NIX_INSTALLER_NO_MODIFY_PROFILE=1

# Set up Nix configuration
RUN mkdir -p /etc/nix
RUN echo "experimental-features = nix-command flakes" >> /etc/nix/nix.conf && \
    echo "filter-syscalls = false" >> /etc/nix/nix.conf && \
    echo "sandbox = false" >> /etc/nix/nix.conf && \
    echo "trusted-substituters = https://cache.nixos.org https://cache.flox.dev" >> /etc/nix/nix.conf && \
    echo "trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY= flox-cache-public-1:7F4OyH7ZCnFhcze3fJdfyXYLQw/aV7GEed86nQ7IsOs=" >> /etc/nix/nix.conf

RUN curl -L https://nixos.org/nix/install | NIX_BUILD_GROUP_ID=122 bash -s -- --daemon  --nix-extra-conf-file /etc/nix/nix.conf

# Copy nix-daemon service script
COPY nix-daemon /etc/init.d/nix-daemon
RUN chmod a+rx /etc/init.d/nix-daemon

# Set up environment
ENV PATH="/nix/var/nix/profiles/default/bin:$PATH"

# Add nix-daemon to default runlevel (but don't start it yet)
RUN rc-update add nix-daemon default

# Install Flox
RUN . /nix/var/nix/profiles/default/etc/profile.d/nix.sh && \
    nix profile install --impure \
    --profile /nix/var/nix/profiles/default \
    --experimental-features "nix-command flakes" \
    --accept-flake-config \
    'github:flox/flox'


# Create a startup script
RUN echo '#!/bin/sh' > /start.sh && \
    echo 'rc-service nix-daemon start' >> /start.sh && \
    echo 'exec "$@"' >> /start.sh && \
    chmod +x /start.sh

# Set the entrypoint to our startup script
ENTRYPOINT ["/start.sh"]

# Set the default command
CMD ["/bin/bash"]
