#!/bin/bash

set -e

build_bacalhau() {
  make build
}


run_bacalhau_test() {
  make bash-test 
}


main () {
  build_bacalhau
  bash-test
}

main


