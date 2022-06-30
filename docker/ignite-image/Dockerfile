FROM weaveworks/ignite-ubuntu
RUN apt-get update -y && \
    apt-get -y install python3-pip fuse && \
    pip3 install --quiet psrecord matplotlib
RUN wget -q https://dist.ipfs.io/go-ipfs/v0.11.0/go-ipfs_v0.11.0_linux-amd64.tar.gz && \
    tar -xvzf go-ipfs_v0.11.0_linux-amd64.tar.gz && \
    cd go-ipfs && \
    bash install.sh
RUN mkdir /ipfs && mkdir /ipns
RUN ls -la go-ipfs
RUN cd go-ipfs && echo $PWD