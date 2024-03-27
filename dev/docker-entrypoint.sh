#!/usr/bin/env bash

go version
eval "$(ssh-agent -s)"
grep -slR "PRIVATE" ~/.ssh/ | xargs ssh-add
poetry install --no-root
source .venv/bin/activate

exec "$@"