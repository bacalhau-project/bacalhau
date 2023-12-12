FROM --platform=$TARGETPLATFORM python:3.11.7-bullseye

RUN mkdir /build
WORKDIR /build

RUN apt-get -yq update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y \
    make \
    build-essential \
    libssl-dev \
    zlib1g-dev \
    libbz2-dev \
    libreadline-dev \
    libsqlite3-dev \
    wget \
    curl \
    llvm \
    libncurses5-dev \
    libncursesw5-dev \
    xz-utils \
    tk-dev \
    libffi-dev \
    liblzma-dev \
    git 

RUN python -mpip install --upgrade pip
RUN python -mpip install poetry 

COPY base_requirements.txt /build
COPY launcher.py /build
RUN python -mpip install -r /build/base_requirements.txt


CMD ["/build/launcher.py"]

LABEL org.opencontainers.image.source https://github.com/bacalhau-project/bacalhau-images
LABEL org.opencontainers.image.title "Bacalhau custom jobtype - Python 3.11"
LABEL org.opencontainers.image.description "Python for the bacalhau custom job type"
LABEL org.opencontainers.image.licenses Apache-2.0
LABEL org.opencontainers.image.url https://bacalhau.org