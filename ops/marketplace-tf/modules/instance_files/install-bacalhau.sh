#!/usr/bin/env bash

set -x

# Usage: install-bacalhau [release <version> | branch <branch-name> | commit <commit-sha>]
function install-bacalhau() {
    if [ $# -eq 0 ]; then
        install-bacalhau-default
        elif [ "$1" = "release" ]; then
        install-bacalhau-from-release "$2"
        elif [ "$1" = "branch" ]; then
        install-bacalhau-from-branch "$2"
        elif [ "$1" == "commit" ]; then
        install-bacalhau-from-commit "$2"
    else
        echo "Invalid argument: $1" >&2
        echo "Usage: install-bacalhau [release <version> | branch <branch-name> | commit <commit-sha>]" >&2
        return 1
    fi
}

# installs bacalhau using the get.bacalhau.org/install.sh script.
function install-bacalhau-default() {
    echo "Installing Bacalhau from get.bacalhau.org"
    curl -sL https://get.bacalhau.org/install.sh | bash || {
        echo "Failed to install Bacalhau using default method" >&2
        return 1
    }
}

# installs bacalhau from a specific release tag, e.g. v1.0.0, v1.2.1, etc.
function install-bacalhau-from-release() {
    local version=$1
    echo "Installing Bacalhau from release ${version}"
    apt-get -y install --no-install-recommends jq || {
        echo "Failed to install jq" >&2
        return 1
    }

    wget "https://github.com/bacalhau-project/bacalhau/releases/download/${version}/bacalhau_${version}_linux_amd64.tar.gz" || {
        echo "Failed to download Bacalhau release ${version}" >&2
        return 1
    }

    tar xfv "bacalhau_${version}_linux_amd64.tar.gz" || {
        echo "Failed to extract Bacalhau release ${version}" >&2
        return 1
    }

    mv ./bacalhau /usr/local/bin/bacalhau || {
        echo "Failed to move Bacalhau to /usr/local/bin" >&2
        return 1
    }
}



# installs bacalhau based on a branch name, e.g. main, frrist/some-code, walid/some-other-code, etc.
function install-bacalhau-from-branch() {
    local branch=$1
    echo "Installing Bacalhau from branch ${branch}"
    install-bacalhau-dependencies || return 1

    git clone --branch "${branch}" https://github.com/bacalhau-project/bacalhau.git || {
        echo "Failed to clone repository" >&2
        return 1
    }

    pushd bacalhau || return 1
    build-bacalhau-from-source || return 1
    popd
}

# installs the version of bacalhau based on a git commit sha.
function install-bacalhau-from-commit() {
    local commit_sha=$1
    echo "Installing Bacalhau from commit ${commit_sha}"
    install-bacalhau-dependencies || return 1

    git clone https://github.com/bacalhau-project/bacalhau.git || {
        echo "Failed to clone repository" >&2
        return 1
    }

    pushd bacalhau || return 1
    git checkout "${commit_sha}" || {
        echo "Failed to checkout commit ${commit_sha}" >&2
        popd
        return 1
    }

    build-bacalhau-from-source || return 1
    popd
}

# Common function to build bacalhau from source
function build-bacalhau-from-source() {
    make || {
        echo "Failed to build bacalhau" >&2
        return 1
    }

    mv ./bin/*/*/bacalhau /usr/local/bin/bacalhau || {
        echo "Failed to move Bacalhau to /usr/local/bin" >&2
        return 1
    }
}

# Main function to install all dependencies
function install-bacalhau-dependencies() {
    echo "Installing Bacalhau dependencies..."
    apt-update
    install-apt-dependencies
    install-golang
    install-earthly
}

# Function to update apt package lists
function apt-update() {
    sudo apt update -y || {
        echo "Failed to update package lists." >&2
        return 1
    }
}

# Function to install dependencies via apt
function install-apt-dependencies() {
    sudo apt-get -y install --no-install-recommends jq make gcc g++ zip vim || {
        echo "Failed to install apt dependencies." >&2
        return 1
    }
}


# Function to install Earthly
function install-earthly() {
    sudo /bin/sh -c 'wget https://github.com/earthly/earthly/releases/latest/download/earthly-linux-amd64 -O /usr/local/bin/earthly && chmod +x /usr/local/bin/earthly && /usr/local/bin/earthly bootstrap --with-autocomplete' || {
        echo "Failed to install Earthly." >&2
        return 1
    }
}

function install-golang() {
    # Install go
    export HOME=/root
    export GOCACHE="$HOME/.cache/go-build"
    export GOPATH="/root/go"
    export PATH="$PATH:$GOPATH/bin:/usr/local/go/bin"
    sudo mkdir -p "$GOPATH"

    sudo rm -fr /usr/local/go /usr/local/bin/go
    curl --silent --show-error --location --fail 'https://go.dev/dl/go1.21.8.linux-amd64.tar.gz' | sudo tar --extract --gzip --file=- --directory=/usr/local
    sudo ln -s /usr/local/go/bin/go /usr/local/bin/go
    go version
}

install-bacalhau "$@"
