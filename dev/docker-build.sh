#!/usr/bin/env bash
# Allow direnv to work in the current shell
eval "$(direnv hook bash)"
direnv allow

# Add the current directory to the safe directory list
git config --global --add safe.directory /workspaces/bacalhau
python3 --version
python3 -m ensurepip
python3 -m pip install --upgrade pip
python3 -m pip install uv
uv venv
. .venv/bin/activate
uv pip install poetry