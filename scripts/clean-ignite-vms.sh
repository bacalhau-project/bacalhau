#!/bin/bash
set -xeuo pipefail
sudo ignite rm -f $(sudo ignite ps -a | tail -n +2 | awk '{print $1}')