#!/bin/bash

function install-bacalhau() {
  if [ $# -eq 0 ]; then
    install-bacalhau-default
  elif [ "$1" = "release" ]; then
    install-bacalhau-from-release "$2"
  elif [ "$1" = "branch" ]; then
    install-bacalhau-from-source "$2"
  else
    echo "Invalid argument: $1" >&2
    echo "Usage: install-bacalhau [release <version> | branch <branch-name>]" >&2
    return 1
  fi
}

function install-bacalhau-default() {
  echo "Installing Bacalhau from get.bacalhau.org"
  curl -sL https://get.bacalhau.org/install.sh | bash || {
    echo "Failed to install Bacalhau using default method" >&2
    return 1
  }
}

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

function install-bacalhau-from-source() {
  local branch=$1
  echo "Installing Bacalhau from branch ${branch}"

  sudo apt-get -y install --no-install-recommends jq nodejs npm make || {
    echo "Failed to install dependencies" >&2
    return 1
  }

  git clone --branch "${branch}" https://github.com/bacalhau-project/bacalhau.git || {
    echo "Failed to clone repository" >&2
    return 1
  }

  pushd bacalhau || return 1
  pushd webui || return 1
  npm install || {
    echo "Failed to install npm packages" >&2
    popd
    return 1
  }
  popd

  make build-bacalhau || {
    echo "Failed to build bacalhau" >&2
    popd
    return 1
  }
  mv ./bin/*/bacalhau /usr/local/bin/bacalhau
  popd
}

install-bacalhau "$@"

