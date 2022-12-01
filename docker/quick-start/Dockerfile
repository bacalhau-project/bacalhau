
# this container should match the manual steps in 
# https://docs.bacalhau.org/running-node/quick-start

# TODO: build-args for
#  export IPFS_VERSION=$(wget -q -O - https://raw.githubusercontent.com/filecoin-project/bacalhau/main/ops/terraform/production.tfvars | grep --color=never ipfs_version | awk -F'"' '{print $2}')

FROM ubuntu:22.04 as ipfs
# NOT USED ANYMORE
RUN apt-get update -y && apt-get install -y wget fuse

RUN wget -q https://dist.ipfs.tech/go-ipfs/v0.12.2/go-ipfs_v0.12.2_linux-amd64.tar.gz && \
    tar -xvzf go-ipfs_v0.12.2_linux-amd64.tar.gz && \
    cd go-ipfs && \
    bash install.sh \
    sudo bash install.sh \
    ipfs --version

VOLUME /data
ENV IPFS_PATH=/data/ipfs

ADD ipfs-entrypoint.sh /entrypoint.sh
ENTRYPOINT ["bash", "/entrypoint.sh"]

FROM ubuntu:22.04 as bacalhau
RUN apt-get update -y && apt-get install -y curl


RUN curl -sL https://get.bacalhau.org/install.sh | bash

# TODO: need the docker binary.. (or another way to inject IPFS_CONNECT)

ADD bacalhau-entrypoint.sh /entrypoint.sh
ENTRYPOINT ["bash", "/entrypoint.sh"]

FROM ipfs as kitchensink
RUN apt-get update -y && apt-get install -y curl


RUN curl -sL https://get.bacalhau.org/install.sh | bash

# TODO: need the docker binary.. (or another way to inject IPFS_CONNECT)

ADD bacalhau-entrypoint.sh /bacalhau-entrypoint.sh
ENTRYPOINT ["bash", "/bacalhau-entrypoint.sh"]