#!/usr/bin/env bash
unset LD_LIBRARY_PATH

export SHELL=$(command -v bash)
. <(FLOX_DISABLE_METRICS=true flox activate -r aronchick/bacalhau -t;);

just build-webui
