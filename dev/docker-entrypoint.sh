#!/usr/bin/env bash

go version
eval "$(ssh-agent -s)"
grep -slR "PRIVATE" ~/.ssh/ | xargs ssh-add
source .venv/bin/activate

exec "$@"