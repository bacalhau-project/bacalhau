
# cribbed from https://github.com/ipfs-cluster/ipfs-cluster/blob/master/Dockerfile
#   similar to https://github.com/ipfs/kubo/blob/master/Dockerfile

FROM golang:1.19-buster AS builder

ENV PROJECT     filecoin-project/bacalhau

ENV GOPATH      /go
ENV SRC_PATH    $GOPATH/src/github.com/$PROJECT
ENV GO111MODULE on
ENV GOPROXY     https://proxy.golang.org

ENV SUEXEC_VERSION v0.2
ENV TINI_VERSION v0.19.0
RUN set -eux; \
    dpkgArch="$(dpkg --print-architecture)"; \
    case "${dpkgArch##*-}" in \
    "amd64" | "armhf" | "arm64") tiniArch="tini-static-$dpkgArch" ;;\
    *) echo >&2 "unsupported architecture: ${dpkgArch}"; exit 1 ;; \
    esac; \
    cd /tmp \
    && git clone https://github.com/ncopa/su-exec.git \
    && cd su-exec \
    && git checkout -q $SUEXEC_VERSION \
    && make su-exec-static \
    && cd /tmp \
    && wget -q -O tini https://github.com/krallin/tini/releases/download/$TINI_VERSION/$tiniArch \
    && chmod +x tini

# Get the TLS CA certificates, they're not provided by busybox.
RUN apt-get update && apt-get install -y ca-certificates curl

# Docker client for bacalhau exec
RUN cd /tmp \
    && curl -sL -o docker.tgz https://download.docker.com/linux/static/stable/x86_64/docker-20.10.21.tgz \
    && tar zxvf docker.tgz \
    && cp /tmp/docker/docker /usr/bin/docker

COPY --chown=1000:users go.* $SRC_PATH/
WORKDIR $SRC_PATH
RUN go mod download

COPY --chown=1000:users . $SRC_PATH
RUN make build-image-dev


#------------------------------------------------------
FROM busybox:1-glibc AS daemon
ENV PROJECT     filecoin-project/bacalhau

ENV GOPATH      /go
ENV SRC_PATH    $GOPATH/src/github.com/$PROJECT

# 127.0.0.1 is local to the container only
ENV BACALHAU_API_HOST=0.0.0.0
ENV BACALHAU_API_PORT=1234

# https://docs.bacalhau.org/running-node/networking
# libp2p swarm port - required for node-to-node
EXPOSE 1235
# REST API port for bacalhau client
EXPOSE ${BACALHAU_API_PORT}
# metrics port
EXPOSE 2112


COPY --from=builder /usr/local/bin/bacalhau /usr/local/bin/bacalhau
COPY --from=builder $SRC_PATH/docker/entrypoint.sh /usr/local/bin/entrypoint.sh

# TODO: consider su-exec to non-root, but this makes accessing the Docker socket more complex
COPY --from=builder /tmp/su-exec/su-exec-static /sbin/su-exec
COPY --from=builder /tmp/tini /sbin/tini
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /usr/bin/docker /usr/bin/docker


ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/entrypoint.sh"]

# initially defaults to use devstack ipfs
CMD ["devstack"]
