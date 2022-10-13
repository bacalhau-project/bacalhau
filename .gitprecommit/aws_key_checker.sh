#!/usr/bin/env bash

if git rev-parse --verify HEAD >/dev/null 2>&1
then
    against=HEAD
else
    # Initial commit: diff against an empty tree object
    EMPTY_TREE=$(git hash-object -t tree /dev/null)
    against=$EMPTY_TREE
fi

# Redirect output to stderr.
exec 1>&2
 
# Check changed files for an AWS keys
FILES=$(git diff --cached --name-only $against)

if [ -n "$FILES" ]; then
    KEY_ID=$(grep -E --line-number '[A-Z0-9]{20}[^A-Z0-9]?' $FILES)
    KEY=$(grep -E --line-number '[A-Za-z0-9/+=]{40}[^A-Za-z0-9/+=]?' $FILES)

    if [ -n "$KEY_ID" ] || [ -n "$KEY" ]; then
        exec < /dev/tty # Capture input
        echo "=========== Possible AWS Access Key IDs ==========="
        echo "${KEY_ID}"
        echo ""

        echo "=========== Possible AWS Secret Access Keys ==========="
        echo "${KEY}"
        echo ""

        while true; do
            read -p "[AWS Key Check] Possible AWS keys found. Commit files anyway? (y/N) " yn
            if [ "$yn" = "" ]; then
                yn='N'
            fi
            case $yn in
                [Yy] ) exit 0;;
                [Nn] ) exit 1;;
                * ) echo "Please answer y or n for yes or no.";;
            esac
        done
        exec <&- # Release input
    fi
fi

# Normal exit
exit 0
