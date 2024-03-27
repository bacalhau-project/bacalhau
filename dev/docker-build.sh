#!/usr/bin/env bash
git config --global --add safe.directory /workspaces/bacalhau
python3 --version
python3 -m ensurepip
python3 -m pip install --upgrade pip
python3 -m pip install uv
uv venv
. .venv/bin/activate
uv pip install poetry
poetry install --no-root