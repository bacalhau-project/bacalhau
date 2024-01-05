#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# generate.sh
#
# Generate the files for the project.
#
# Usage:
#   generate.sh

go get -u golang.org/x/tools/cmd/stringer
go install golang.org/x/tools/cmd/stringer

go get -u github.com/golang/mock/mockgen
go install github.com/golang/mock/mockgen

go generate ./...
