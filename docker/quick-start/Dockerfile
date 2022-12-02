
# this container should match the manual steps in 
# https://docs.bacalhau.org/running-node/quick-start

# the idea of the quickstart is to help someone bring up the simplest global cluster node

# TODO: build-args for what version of bacalhau?

FROM ubuntu:22.04 as bacalhau
RUN apt-get update -y && apt-get install -y curl

RUN curl -sL https://get.bacalhau.org/install.sh | bash

# TODO: need the docker binary.. (or another way to inject IPFS_CONNECT)
# crib from, or use https://github.com/filecoin-project/bacalhau/blob/b2120012ea/ops/terraform/remote_files/scripts/install-node.sh
# for other deps...

ADD bacalhau-entrypoint.sh /entrypoint.sh
ENTRYPOINT ["bash", "/entrypoint.sh"]

