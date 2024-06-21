#!/usr/bin/env bash
pip3 install poetry
poetry install
poetry run pre-commit install
