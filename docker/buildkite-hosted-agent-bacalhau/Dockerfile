FROM buildkite/hosted-agent-base

ENV GOLANG_VERSION 1.21.0

RUN apt-get update && apt-get install -y wget

# Set environment variables for Go
RUN wget https://golang.org/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GOLANG_VERSION}.linux-amd64.tar.gz \
    && rm go${GOLANG_VERSION}.linux-amd64.tar.gz

# Set Go environment variables
ENV PATH /usr/local/go/bin:$PATH

# Install earthly
RUN wget https://github.com/earthly/earthly/releases/latest/download/earthly-linux-amd64 -O /usr/local/bin/earthly && chmod +x /usr/local/bin/earthly

# Install Python 3.10
RUN apt-get update && apt-get install -y \
    python3.10 \
    python3.10-venv \
    python3.10-dev \
    && rm -rf /var/lib/apt/lists/*

# Set Python 3.10 as the default python3
RUN update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.10 1

# Verify installations
RUN go version && python3 --version

