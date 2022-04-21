FROM ubuntu:20.04
RUN apt-get update -y && apt-get install -y wget fuse
RUN wget https://dist.ipfs.io/go-ipfs/v0.11.0/go-ipfs_v0.11.0_linux-amd64.tar.gz && \
    tar -xvzf go-ipfs_v0.11.0_linux-amd64.tar.gz && \
    cd go-ipfs && \
    bash install.sh
ADD entrypoint.sh /entrypoint.sh
ENTRYPOINT ["bash", "/entrypoint.sh"]