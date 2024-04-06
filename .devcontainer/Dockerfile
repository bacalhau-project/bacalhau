ARG GO_VERSION
FROM golang:${GO_VERSION}-bullseye

ARG USERNAME=vscode

COPY .tool-versions /tmp/.tool-versions
COPY pyproject.toml /tmp/pyproject.toml
COPY poetry.lock /tmp/poetry.lock

RUN apt update
RUN apt install -y wget git gcc curl direnv make zlib1g-dev libbz2-dev \
    libreadline-dev libsqlite3-dev libssl-dev libffi-dev \
    lzma liblzma-dev libbz2-dev zsh sudo

ENV SHELL /usr/bin/zsh

COPY custom-scripts/devcontainer-init.sh /usr/local/share/devcontainer-init.sh
RUN chmod guo+x /usr/local/share/devcontainer-init.sh

ENTRYPOINT [ "/usr/bin/zsh", "/usr/local/share/devcontainer-init.sh" ]
CMD [ "sleep", "infinity" ]