# build container stage
FROM golang:1.18.1-buster AS build-env

# branch or tag of the lotus version to build
ARG BRANCH

RUN echo "Building lotus using 2 KiB sectors from branch $BRANCH in local network"

RUN apt-get update -y && \
    apt-get install sudo cron git mesa-opencl-icd gcc bzr jq pkg-config clang libhwloc-dev ocl-icd-opencl-dev build-essential hwloc -y

ENV LOTUS_PATH="~/.lotus-local-net"
ENV LOTUS_MINER_PATH="~/.lotus-miner-local-net"
ENV CGO_CFLAGS="-D__BLST_PORTABLE__"
ENV CGO_CFLAGS_ALLOW="-D__BLST_PORTABLE__"
ENV LOTUS_SKIP_GENESIS_CHECK="_yes_"
ENV RUSTFLAGS="-C target-cpu=native -g"
ENV FFI_BUILD_FROM_SOURCE=1
ENV NETWORK="2k"

RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

RUN git clone https://github.com/filecoin-project/lotus.git --depth 1 --branch $BRANCH && \
    cd lotus && \
    git submodule update --init --recursive && \
    make clean && \
    make $NETWORK && \
    install -C ./lotus /usr/local/bin/lotus && \
    install -C ./lotus-gateway /usr/local/bin/lotus-gateway && \
    install -C ./lotus-shed /usr/local/bin/lotus-shed && \
    install -C ./lotus-miner /usr/local/bin/lotus-miner && \
    install -C ./lotus-seed /usr/local/bin/lotus-seed

# runtime container stage
FROM ubuntu:20.04
ENV DEBIAN_FRONTEND noninteractive
ENV LOTUS_USER="lotus_user"
#creating cron job to check lotus sync status and restart it if process is killed
RUN  apt-get update && \
     apt-get install git curl nano wget libhwloc-dev -y && \
     rm -rf /var/lib/apt/lists/*

RUN wget -c -q https://golang.org/dl/go1.17.9.linux-amd64.tar.gz -O - | tar -xz -C /usr/local

# create nonroot user and lotus folder
RUN adduser --uid 2000 --gecos "" --disabled-password --quiet $LOTUS_USER
RUN echo 'export PATH=$PATH:/usr/local/go/bin' >> /home/$LOTUS_USER/.bashrc
RUN echo 'export GOLOG_OUTPUT=FILE' >> /home/$LOTUS_USER/.bashrc
RUN echo 'export GOLOG_FILE="/home/$LOTUS_USER/miner.log"' >> /home/$LOTUS_USER/.bashrc

COPY --from=build-env /usr/local/bin/lotus /usr/local/bin/lotus
COPY --from=build-env /usr/local/bin/lotus-gateway /usr/local/bin/lotus-gateway
COPY --from=build-env /usr/local/bin/lotus-shed /usr/local/bin/lotus-shed
COPY --from=build-env /usr/local/bin/lotus-miner /usr/local/bin/lotus-miner
COPY --from=build-env /usr/local/bin/lotus-seed /usr/local/bin/lotus-seed
COPY --from=build-env /etc/ssl/certs /etc/ssl/certs
COPY --from=build-env /lib/x86_64-linux-gnu /lib/
# lotus libraries
COPY --from=build-env   /lib/x86_64-linux-gnu/libutil.so.1 \
                        /lib/x86_64-linux-gnu/librt.so.1 \
                        /lib/x86_64-linux-gnu/libgcc_s.so.1 \
                        /lib/x86_64-linux-gnu/libdl.so.2 \
                        /usr/lib/x86_64-linux-gnu/libltdl.so.7 \
                        /usr/lib/x86_64-linux-gnu/libnuma.so.1 \
                        /usr/lib/x86_64-linux-gnu/libhwloc.so.5 /lib/
COPY --from=build-env   /usr/lib/x86_64-linux-gnu/libOpenCL.so.1.0.0 /lib/libOpenCL.so.1

ENV LOTUS_PATH="/home/$LOTUS_USER/.lotus-local-net"
ENV LOTUS_MINER_PATH="/home/$LOTUS_USER/.lotus-miner-local-net"
ENV LOTUS_SKIP_GENESIS_CHECK="_yes_"
ENV CGO_CFLAGS_ALLOW="-D__BLST_PORTABLE__"
ENV CGO_CFLAGS="-D__BLST_PORTABLE__"
ARG BRANCH

# jq libraries
COPY --from=build-env   /usr/lib/x86_64-linux-gnu/libjq.so.1 /usr/lib/x86_64-linux-gnu/
COPY --from=build-env /usr/lib/x86_64-linux-gnu/libonig.so.5.0.0 /usr/lib/x86_64-linux-gnu/libonig.so.5


# copy jq, script/config files
COPY --from=build-env /usr/bin/jq /usr/bin/
COPY config/config.toml /home/$LOTUS_USER/config.toml
COPY scripts/bash-config scripts/configure scripts/run scripts/healthcheck /etc/lotus/docker/
RUN chown 2000:2000 -R /etc/lotus/docker/ /home/$LOTUS_USER/config.toml
RUN chmod +x /etc/lotus/docker/*

USER $LOTUS_USER
RUN lotus fetch-params 2048 && \
    lotus-seed pre-seal --sector-size 2KiB --num-sectors 2
RUN lotus-seed genesis new ~/localnet.json
RUN lotus-seed genesis add-miner ~/localnet.json ~/.genesis-sectors/pre-seal-t01000.json


# API port
EXPOSE 1234/tcp

# P2P port
EXPOSE 1235/tcp

ENTRYPOINT ["/etc/lotus/docker/run"]