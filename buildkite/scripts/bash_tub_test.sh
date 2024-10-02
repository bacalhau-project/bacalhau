#!/bin/bash

set -e

build_bacalhau() {
  make build
}


bash_test() {
  make bash-test
}


main () {
  build_bacalhau
  bash_test
}

main


