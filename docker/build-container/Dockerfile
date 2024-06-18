FROM debian:bookworm-slim as build

RUN apt update && apt install -y curl sudo xz-utils

RUN curl -sSL https://flox.dev/downloads/debian-archive/flox.x86_64-linux.deb -o /tmp/flox.deb

RUN dpkg -i /tmp/flox.deb

WORKDIR /src

RUN flox pull aronchick/bacalhau
