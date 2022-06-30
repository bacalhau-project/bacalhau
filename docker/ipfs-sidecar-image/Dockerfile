FROM ubuntu:20.04
RUN apt-get update -y && apt-get install -y wget fuse
RUN wget -q https://dist.ipfs.io/go-ipfs/v0.12.2/go-ipfs_v0.12.2_linux-amd64.tar.gz && \
    tar -xvzf go-ipfs_v0.12.2_linux-amd64.tar.gz && \
    cd go-ipfs && \
    bash install.sh
ADD entrypoint.sh /entrypoint.sh
ENTRYPOINT ["bash", "/entrypoint.sh"]