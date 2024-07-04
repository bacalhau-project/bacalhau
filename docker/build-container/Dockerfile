FROM ghcr.io/flox/flox:v1.1.0

# ENV USER_ID=1000
# ENV GROUP_ID=1000

# RUN groupadd -g $GROUP_ID buildkite-agent && \
#     useradd -m -s /bin/bash -u $USER_ID -g buildkite-agent buildkite-agent

# USER buildkite-agent

# Set up environment
ENV PATH="/nix/var/nix/profiles/default/bin:$PATH"
ENV NIX_PROFILES="/nix/var/nix/profiles/default"

# RUN ls -al

# RUN ls -al /

# RUN whoami

# Set working directory
WORKDIR /workdir

# Set git safe directory
RUN git config --global --add safe.directory /workdir

# # Make sure the buildkite-agent has the right permissions
# RUN sudo chown -R buildkite-agent:buildkite-agent /workdir

# Activate Flox environment
RUN . /nix/var/nix/profiles/default/etc/profile.d/nix.sh && \
    bash -c ". <(FLOX_DISABLE_METRICS=true flox activate -vv -r aronchick/bacalhau -t)"

# Set the default command
CMD ["/nix/var/nix/profiles/default/bin/bash"]
