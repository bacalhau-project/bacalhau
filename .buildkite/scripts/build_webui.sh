#!/usr/bin/env bash

unset LD_LIBRARY_PATH

shell=$(command -v bash)
FLOX_DISABLE_METRICS=true SHELL=$shell flox activate -r aronchick/bacalhau -t -- just build-webui
