FROM docker.io/nixos/nix:latest

# Enable nix-command feature and disable sandbox
RUN echo "experimental-features = nix-command flakes" >> /etc/nix/nix.conf && \
    echo "sandbox = false" >> /etc/nix/nix.conf

# Set up Nix configuration
RUN echo "trusted-substituters = https://cache.flox.dev" >> /etc/nix/nix.conf && \
    echo "trusted-public-keys = flox-cache-public-1:7F4OyH7ZCnFhcze3fJdfyXYLQw/aV7GEed86nQ7IsOs=" >> /etc/nix/nix.conf


RUN echo "trusted-substituters = https://cache.nixos.org https://cache.flox.dev" >> /etc/nix/nix.conf && \
    echo "trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY= flox-cache-public-1:7F4OyH7ZCnFhcze3fJdfyXYLQw/aV7GEed86nQ7IsOs=" >> /etc/nix/nix.conf

# Install flox
RUN nix profile install --impure \
    --profile /nix/var/nix/profiles/default \
    --experimental-features "nix-command flakes" \
    --accept-flake-config \
    'github:flox/flox'

# Set up environment
ENV PATH="/nix/var/nix/profiles/default/bin:$PATH"

# Set working directory
WORKDIR /src

RUN export SHELL=$(which $(echo $0))

# Activate Flox environment
RUN . /nix/var/nix/profiles/default/etc/profile.d/nix.sh && \
    . <(FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau -t; );

# # Add Flox to .bashrc
# RUN echo "source /nix/var/nix/profiles/default/etc/profile.d/nix.sh" >> /root/.bashrc && \
#     echo "source /etc/profile.d/flox.sh" >> /root/.bashrc

# # Health check for Flox
# RUN flox --version || echo "Flox installation failed"

# # Set the default command
# CMD ["/nix/var/nix/profiles/default/bin/bash"]

# COPY ./buildkite-agent.cfg /buildkite/buildkite-agent.cfg
# COPY ./buildkite-agent-linux-amd64 /usr/local/bin/buildkite-agent
# COPY ./entrypoint.sh /usr/local/bin/buildkite-agent-entrypoint

# WORKDIR /src

# # Run flox activate
# RUN . /nix/var/nix/profiles/default/etc/profile.d/nix.sh && \
#     FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau -t

# # Set the default command
# ENTRYPOINT ["buildkite-agent-entrypoint"]
