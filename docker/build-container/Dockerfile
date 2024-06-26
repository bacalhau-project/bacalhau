FROM docker.io/bacalhauproject/nix-flox-base:0.0.50

# Set up environment
ENV PATH="/nix/var/nix/profiles/default/bin:$PATH"
ENV NIX_PROFILES="/nix/var/nix/profiles/default"

# Set working directory
WORKDIR /src

RUN export SHELL=$(which bash)

# Activate Flox environment
RUN . /nix/var/nix/profiles/default/etc/profile.d/nix.sh && \
    bash -c ". <(FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau -t)"

# Set the default command
CMD ["/nix/var/nix/profiles/default/bin/bash"]
