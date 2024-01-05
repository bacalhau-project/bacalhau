FROM --platform=$TARGETPLATFORM ubuntu:noble-20231126.1
ARG TARGETPLATFORM
RUN echo "I am building for $TARGETPLATFORM" > /log

RUN apt update && apt -yq upgrade && apt -yq install wget unzip 
RUN if [ $TARGETPLATFORM = 'linux/arm64' ]; then \
    wget -O /tmp/ddb.zip "https://github.com/duckdb/duckdb/releases/download/v0.9.2/duckdb_cli-linux-aarch64.zip"; \
    else \
    wget -O /tmp/ddb.zip "https://github.com/duckdb/duckdb/releases/download/v0.9.2/duckdb_cli-linux-amd64.zip"; \
    fi
RUN unzip /tmp/ddb.zip -d /usr/local/bin

LABEL org.opencontainers.image.source https://github.com/bacalhau-project/bacalhau-images
LABEL org.opencontainers.image.title "Bacalhau custom jobtype - Duckdb"
LABEL org.opencontainers.image.description "Duckdb for the bacalhau custom job type"
LABEL org.opencontainers.image.licenses Apache-2.0
LABEL org.opencontainers.image.url https://bacalhau.org