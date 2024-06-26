set shell := ["bash", "-c"]
set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]
set allow-duplicate-recipes
set positional-arguments
set dotenv-load
set export

SOURCES:="bacalhau_sdk"

build:
    #!/usr/bin/env bash
    cd webui
    yarn install
    yarn build