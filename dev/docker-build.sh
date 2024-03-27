#!/usr/bin/env bash
# Allow direnv to work in the current shell
eval "$(direnv hook bash)"
echo "source .venv/bin/activate" >> .envrc
direnv allow

# Add the current directory to the safe directory list
git config --global --add safe.directory /workspaces/bacalhau
cd /workspaces/bacalhau

python3 --version
python3 -m ensurepip
python3 -m pip install --upgrade pip
python3 -m pip install uv
python3 -m venv .venv
. .venv/bin/activate
python3 -m pip install poetry
python3 -m poetry install --no-root