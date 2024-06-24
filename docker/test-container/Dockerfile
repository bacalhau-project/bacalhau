FROM --platform=linux/amd64 bacalhauproject/build-container:0.0.32

RUN BACALHAU_ENVIRONMENT=test
RUN curl -o /tmp/ipfs.tar.gz "https://dist.ipfs.tech/go-ipfs/v0.18.0/go-ipfs_v0.18.0_linux-amd64.tar.gz"
RUN tar -xzf /tmp/ipfs.tar.gz -C /tmp
RUN cd /tmp/go-ipfs
RUN bash /tmp/go-ipfs/install.sh
RUN ipfs --version
RUN ipfs init
RUN ipfs config Addresses.API /ip4/127.0.0.1/tcp/5001
RUN ipfs config Addresses.Gateway /ip4/0.0.0.0/tcp/8080
RUN echo '~~~ Start IPFS Daemon'
RUN ( ipfs daemon ) &
RUN echo '~~~ Set IPFS Connect Env'
RUN export BACALHAU_NODE_IPFS_CONNECT=/ip4/127.0.0.1/tcp/5001
