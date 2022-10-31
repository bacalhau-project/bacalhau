#!/bin/bash
if git rev-parse --verify HEAD >/dev/null 2>&1
then
    against=HEAD
else
    # Initial commit: diff against an empty tree object
    EMPTY_TREE=$(git hash-object -t tree /dev/null)
    against=${EMPTY_TREE}
fi

FILES=$(git diff --cached --diff-filter=d --name-only "${against}")
if [[ -n "${FILES}" ]]; then
    for f in ${FILES}; do
        if [[ "${f}" == *.go ]]; then
            gofmt -w -s "${f}"
        fi
    done
fi