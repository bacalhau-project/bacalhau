#!/bin/bash
python3 -q -m pip install --upgrade pip
pip3 install poetry
poetry install
poetry run pre-commit install

pushd python || exit
poetry install
popd || exit

pushd webui || exit
npm install
popd || exit
