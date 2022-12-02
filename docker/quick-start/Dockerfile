
# this container should match the manual steps in 
# https://docs.bacalhau.org/running-node/quick-start

# the idea of the quickstart is to help someone bring up the simplest global cluster node

# TODO: build-args for what version of bacalhau?

FROM ubuntu:22.04 as bacalhau

# crib from, or use https://github.com/filecoin-project/bacalhau/blob/b2120012ea/ops/terraform/remote_files/scripts/install-node.sh
RUN apt-get update -y && apt-get install -y \
      ca-certificates \
      curl \
      gnupg \
      lsb-release

RUN curl -sL https://get.bacalhau.org/install.sh | bash

RUN cd /tmp \
    && curl -sL -o docker.tgz https://download.docker.com/linux/static/stable/x86_64/docker-20.10.21.tgz \
    && tar zxvf docker.tgz \
    && cp /tmp/docker/docker /usr/bin/docker

ADD bacalhau-entrypoint.sh /entrypoint.sh
ENTRYPOINT ["bash", "/entrypoint.sh"]

