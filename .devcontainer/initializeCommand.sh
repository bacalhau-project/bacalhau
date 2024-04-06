#!/usr/bin/env bash

files_to_copy=(
    ".tool-versions"
    "pyproject.toml"
    "poetry.lock"
)

for file in "${files_to_copy[@]}"; do
    cp "${file}" ".devcontainer/${file}"
done