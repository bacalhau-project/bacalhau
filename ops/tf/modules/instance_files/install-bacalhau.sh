#!/bin/bash

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

# installs the dependencies required to build bacalhau from source
function install-bacalhau-dependencies() {
  echo "Installing Bacalhau dependencies..."
  sudo apt update -y
  sudo apt-get -y install --no-install-recommends jq make gcc g++ zip || {
    echo "Failed to install dependencies." >&2
    return 1
  }

  curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
  sudo apt-get install -y nodejs

  sudo apt remove cmdtest -y
  curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -
  echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
  sudo apt update
  sudo apt-get -y install --no-install-recommends yarn
}

# installs bacalhau based on a branch name, e.g. main, frrist/some-code, simon/some-other-code, etc.
function install-bacalhau-from-branch() {
  local branch=$1
  echo "Installing Bacalhau from branch ${branch}"
  install-bacalhau-dependencies || return 1

  git clone --branch "${branch}" https://github.com/bacalhau-project/bacalhau.git || {
    echo "Failed to clone repository" >&2
    return 1
  }

  pushd bacalhau || return 1
  pushd webui || return 1
  yarn install || {
    echo "Failed to install yarn packages" >&2
    popd
    return 1
  }
  popd

  make build-bacalhau || {
    echo "Failed to build bacalhau" >&2
    popd
    return 1
  }
  mv ./bin/*/*/bacalhau /usr/local/bin/bacalhau
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

  pushd webui || return 1
  yarn install || {
    echo "Failed to install yarn packages" >&2
    popd
    return 1
  }
  popd

  make build-bacalhau || {
    echo "Failed to build bacalhau" >&2
    popd
    return 1
  }
  mv ./bin/*/*/bacalhau /usr/local/bin/bacalhau
  popd
}

install-bacalhau "$@"