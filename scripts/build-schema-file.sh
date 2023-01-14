#!/bin/sh
set -euo pipefail
IFS=$'\n\t'

TARGET_TAG=$1
TARGET_COMMIT=$(git rev-parse $TARGET_TAG)
CURRENT_COMMIT=$(git rev-parse HEAD)
  
if [ $TARGET_COMMIT != $CURRENT_COMMIT ]; then
    mkdir $TARGET_TAG
    pushd $TARGET_TAG 1>&2

    git clone .. . 1>&2
    git checkout $TARGET_TAG 1>&2
fi

go run . validate --output-schema

if [ $TARGET_COMMIT != $CURRENT_COMMIT ]; then
    popd 1>&2
    rm -rf $TARGET_TAG
fi
