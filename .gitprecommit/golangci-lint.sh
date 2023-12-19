#!/bin/bash
if git rev-parse --verify HEAD >/dev/null 2>&1
then
    against=HEAD
else
    # Initial commit: diff against an empty tree object
    EMPTY_TREE=$(git hash-object -t tree /dev/null)
    against=${EMPTY_TREE}
fi

FILES=$(git diff --cached --name-only "${against}")
if [[ -n "${FILES}" ]]; then
    golangci-lint run --allow-parallel-runners --fix
fi