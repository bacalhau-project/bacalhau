FROM ubuntu:20.04

# Set environment variable for headless installation
ENV DEBIAN_FRONTEND=noninteractive

# Install sudo
RUN apt-get update && apt-get install -y sudo

# Make Bash the default shell
SHELL ["/bin/bash", "-c"]

ENV USER_ID=1000
ENV GROUP_ID=1000

RUN groupadd -g $GROUP_ID buildkite-agent && \
    useradd -m -s /bin/bash -u $USER_ID -g buildkite-agent buildkite-agent

# Add buildkite-agent to sudoers
RUN echo "buildkite-agent ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Install required packages
RUN sudo apt-get update && sudo apt-get install -y \
    bash \
    curl \
    xz-utils \
    git \
    git-lfs \
    && rm -rf /var/lib/apt/lists/*

RUN git config --global --add safe.directory '*'

# Install Nix
ENV NIX_INSTALLER_EXTRA_CONF='filter-syscalls = false'
ENV NIX_INSTALLER_NO_MODIFY_PROFILE=1

# Set up Nix configuration
RUN mkdir -p /etc/nix && \
    echo "experimental-features = nix-command flakes" >> /etc/nix/nix.conf && \
    echo "filter-syscalls = false" >> /etc/nix/nix.conf && \
    echo "sandbox = false" >> /etc/nix/nix.conf && \
    echo "trusted-substituters = https://cache.nixos.org https://cache.flox.dev" >> /etc/nix/nix.conf && \
    echo "trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY= flox-cache-public-1:7F4OyH7ZCnFhcze3fJdfyXYLQw/aV7GEed86nQ7IsOs=" >> /etc/nix/nix.conf

RUN curl -L https://nixos.org/nix/install | NIX_BUILD_GROUP_ID=122 bash -s -- --daemon --nix-extra-conf-file /etc/nix/nix.conf

# Set up environment
ENV PATH="/nix/var/nix/profiles/default/bin:$PATH"

USER buildkite-agent

RUN /bin/bash -c '. /nix/var/nix/profiles/default/etc/profile.d/nix.sh && nix profile install --impure --profile /nix/var/nix/profiles/default --experimental-features "nix-command flakes" --accept-flake-config github:flox/flox'


# Install Flox
RUN /bin/bash -c '\
    . /nix/var/nix/profiles/default/etc/profile.d/nix.sh && \
    nix profile install --impure \
    --profile /nix/var/nix/profiles/default \
    --experimental-features "nix-command flakes" \
    --accept-flake-config \
    github:flox/flox'

# Copy the startup script
COPY start.sh /start.sh
RUN chmod ugo+x /start.sh
# Set the entrypoint to our startup script
ENTRYPOINT ["/start.sh"]

# Set the default command
CMD ["/bin/bash"]
