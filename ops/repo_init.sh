#!/usr/bin/env bash
python3 -q -m pip install --upgrade pip
pip3 install poetry
poetry install
poetry run pre-commit install

pushd python || exit
poetry install
popd || exit

pushd webui || exit
corepack --yes enable
yarn install
popd || exit
