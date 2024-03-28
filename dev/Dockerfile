FROM python:3.11.8-slim

# Install Go
ARG GO_VERSION
ARG GO_ARCHIVE_NAME

RUN apt-get update
RUN apt-get install -y wget git gcc curl direnv make

# Install asdf
RUN git clone https://github.com/asdf-vm/asdf.git ~/.asdf --branch v0.14.0 && \
    echo ". $HOME/.asdf/asdf.sh" >> /root/.bashrc && \
    echo ". $HOME/.asdf/asdf.sh" >> /root/.zshrc

SHELL ["/bin/bash", "-lc"]

# Install all packages from .tool-versions
COPY .tool-versions /tmp/.tool-versions

# Cut each line and install the package. The first time, it's the package, then it's the package and the version
RUN cat /tmp/.tool-versions | while read line; do \
    package=$(echo $line | cut -d ' ' -f 1); \
    version=$(echo $line | cut -d ' ' -f 2); \
    asdf plugin add $package; \
    asdf install $package $version; \
    asdf global $package $version; \
    done

# Install git-town follow redirect
RUN curl -L https://git-town.com/install.sh -o /tmp/git-town-install.sh && \
    chmod +x /tmp/git-town-install.sh && \
    /tmp/git-town-install.sh

RUN mkdir -p /workspaces/bacalhau
WORKDIR /workspaces/bacalhau
COPY pyproject.toml poetry.lock /workspaces/bacalhau/

# Do the below inside the container so that they are using the right .venv
COPY ./docker-build.sh /
RUN chmod +x /docker-build.sh
RUN /docker-build.sh

COPY ./docker-entrypoint.sh /
RUN chmod +x /docker-entrypoint.sh
ENTRYPOINT [ "/docker-entrypoint.sh" ]
CMD [ "sleep", "infinity" ]