#!/usr/bin/env bash
#
# Copyright 2024 The Bacalhau Authors
#

# Copyright 2021 The Dapr Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
#
# Initializes the devcontainer tasks each time the container starts.
# Users can edit this copy under /usr/local/share in the container to
# customize this as needed for their custom localhost bindings.

set -e
echo "Running devcontainer-init.sh ..."

go version
eval "$(ssh-agent -s)"
grep -slR "PRIVATE" ~/.ssh/ | xargs ssh-add

# Invoke /usr/local/share/docker-bind-mount.sh or docker-init.sh as appropriate
set +e
if [[ "${BIND_LOCALHOST_DOCKER,,}" == "true" ]]; then
    echo "Invoking docker-bind-mount.sh ..."
    exec /usr/local/share/docker-bind-mount.sh "$@"
else
    echo "Invoking docker-init.sh ..."
    exec /usr/local/share/docker-init.sh "$@"
fi