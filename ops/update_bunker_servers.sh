#!/bin/bash
set -xeuo pipefail

for NODE in {mind,node0{1..4}}.lukemarsden.net; do
    ssh luke@$NODE -- "curl -sL https://get.bacalhau.org/install.sh | bash"
    ssh luke@$NODE -- pkill bacalhau
done
