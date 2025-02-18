FROM docker.io/buildkite/hosted-agent-base:ubuntu-v1.0.0@sha256:af0d05215252cc0d362135fcf1f3413e6a5a2ff11e8c40d8b8ba453f4d7f245a

# When updatin the golang version golangci-lint may also need its version updated for compatibility.
ENV GOLANG_VERSION 1.23.0
ENV GOLANGCI_LINT_VERSION v1.64.2

RUN apt-get update && apt-get install -y wget make iproute2 gh

# Set environment variables for Go
RUN wget https://golang.org/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GOLANG_VERSION}.linux-amd64.tar.gz \
    && rm go${GOLANG_VERSION}.linux-amd64.tar.gz

# Set Go environment variables
ENV PATH /usr/local/go/bin:$PATH
ENV GOPATH /usr/local/go
ENV GO111MODULE on
ENV CGO_ENABLED 0


# Install earthly
RUN wget https://github.com/earthly/earthly/releases/latest/download/earthly-linux-amd64 -O /usr/local/bin/earthly && chmod +x /usr/local/bin/earthly

# Install golanci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin ${GOLANGCI_LINT_VERSION}

# Install Python 3.10
RUN apt-get update && apt-get install -y \
    python3.10 \
    python3.10-venv \
    python3.10-dev \
    && rm -rf /var/lib/apt/lists/*

# Set Python 3.10 as the default python3
RUN update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.10 1

# Install Pre Commit
RUN pip install pre-commit

# Install GoTestSum
RUN go install gotest.tools/gotestsum@v1.8.2

# Install Swag for Swagger Generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Verify installations
RUN go version && python3 --version

# Install IPFS
ENV ipfs_version v0.18.0


# Install IPFS
RUN wget https://dist.ipfs.tech/go-ipfs/${ipfs_version}/go-ipfs_${ipfs_version}_linux-amd64.tar.gz -O ipfs.tar.gz \
 && tar -xzf ipfs.tar.gz \
 && cd go-ipfs \
 && sudo bash install.sh \
 && ipfs --version
