# Based on https://github.com/dapr/dapr/blob/master/docker/Dockerfile-dev

# [Choice] Go version: 1, 1.21, etc
# Install Go
ARG GO_VERSION
FROM golang:${GO_VERSION}-bullseye

# [Option] Install zsh
ARG INSTALL_ZSH="true"

# This Dockerfile adds a non-root 'bacalhau' user with sudo access. However, for Linux,
# this user's GID/UID must match your local user UID/GID to avoid permission issues
# with bind mounts. Update USER_UID / USER_GID if yours is not 1000. See
# https://aka.ms/vscode-remote/containers/non-root-user for details.
ARG USERNAME=bacalhau
ARG USER_UID=1001
ARG USER_GID=$USER_UID

# Other env vars
ENV GO111MODULE=auto
ENV CGO_ENABLED=0
ENV DOCKER_BUILDKIT=1
ENV ASDF_DIR=/home/${USERNAME}/.asdf

RUN apt update
RUN apt install -y wget git gcc curl direnv make zlib1g-dev libbz2-dev \
    libreadline-dev libsqlite3-dev libssl-dev libffi-dev \
    lzma liblzma-dev libbz2-dev zsh sudo

RUN useradd -ms /usr/bin/zsh ${USERNAME}
USER ${USERNAME}


RUN git config --global advice.detachedHead false
RUN git clone https://github.com/asdf-vm/asdf.git ${ASDF_DIR} --branch v0.14.0
RUN echo ". ${ASDF_DIR}/asdf.sh" >> /home/${USERNAME}/.zshrc
RUN echo "export PATH=${ASDF_DIR}/bin:${PATH}" >> /home/${USERNAME}/.zshrc
ENV PATH="${ASDF_DIR}/bin:${ASDF_DIR}/shims:${PATH}"

# Install all packages from .tool-versions
COPY .tool-versions /tmp/.tool-versions

# Setup image using library scripts and configure non-root user.
COPY library-scripts/* custom-scripts/* first-run-notice.txt /tmp/staging/

USER ${USERNAME}
#
# Install asdf plugins & tools
RUN bash /tmp/staging/install-asdf-tools.sh "/tmp/.tool-versions"

USER root

RUN bash /tmp/staging/setup-user.sh

RUN bash /tmp/staging/common-debian.sh "${INSTALL_ZSH}" "${USERNAME}" "${USER_UID}" "${USER_GID}" "true" "true" "true"

# Copy our init scripts to /usr/local/share.
RUN mv -f -t /usr/local/share/ /tmp/staging/devcontainer-init.sh \
    && chmod +x /usr/local/share/devcontainer-init.sh \
    && chown ${USERNAME}:root /usr/local/share/devcontainer-init.sh \
    #
    # Move the first run notice to the correct location for Codespaces.
    && mkdir -p /usr/local/etc/vscode-dev-containers/ \
    && mv -f /tmp/staging/first-run-notice.txt /usr/local/etc/vscode-dev-containers/

#
# Set permissions for the workspace folder
RUN mkdir -p /workspaces && chown ${USERNAME} /workspaces \
    && mkdir -p /workspaces/${USERNAME} \
    && chown -R ${USERNAME} /workspaces/${USERNAME}

USER ${USERNAME}

COPY pyproject.toml /tmp/pyproject.toml

#
# Install Python and pip.
RUN bash /tmp/staging/install-python-and-packages.sh "${USERNAME}" "/workspaces/${USERNAME}" "/tmp/pyproject.toml"

USER root

#
# Clean up packages and the staging folder.
RUN apt-get autoremove -y && apt-get clean -y && rm -rf /tmp/staging

# Mount for docker-in-docker
VOLUME [ "/var/lib/docker" ] 

# Initialize Dapr devcontainer script
ENTRYPOINT [ "/usr/local/share/devcontainer-init.sh" ]
CMD [ "sleep", "infinity" ]

USER ${USERNAME}